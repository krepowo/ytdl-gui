//go:build windows

package ytdlp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// CREATE_NO_WINDOW keeps a console window from flashing when yt-dlp runs.
const createNoWindow = 0x08000000

// ntdll's NtSuspendProcess/NtResumeProcess implement a true pause (the process
// keeps its state and file handles, and resumes from the same byte offset).
// They are undocumented but stable and widely used; if a future Windows drops
// them, Call fails and the job falls back to kill + --continue.
var (
	modntdll             = windows.NewLazySystemDLL("ntdll.dll")
	procNtSuspendProcess = modntdll.NewProc("NtSuspendProcess")
	procNtResumeProcess  = modntdll.NewProc("NtResumeProcess")
)

// jobBasicProcessIDList mirrors JOBOBJECT_BASIC_PROCESS_ID_LIST, which x/sys
// does not export. The list is a fixed-size array plus a count.
type jobBasicProcessIDList struct {
	NumberOfAssignedProcesses uint32
	NumberOfProcessIDsInList  uint32
	ProcessIDList             [1024]uintptr
}

// winChild is a real yt-dlp process managed through a Windows Job Object. The
// job guarantees that every child the process spawns (notably ffmpeg.exe and the
// PyInstaller python.exe child) is terminated with it, so a cancel/quit never
// leaves orphans.
type winChild struct {
	cmd *exec.Cmd
	job windows.Handle

	r *os.File // read end of the merged stdout+stderr pipe
	w *os.File // write end, closed by the parent once the child is started

	mu     sync.Mutex
	waited bool
	err    error
}

// spawnProcess starts name with args under a new Job Object. stdout and stderr
// are merged into one stream because yt-dlp prints progress to stderr.
func spawnProcess(ctx context.Context, name string, args []string) (child, error) {
	job, err := createKillOnCloseJob()
	if err != nil {
		return nil, fmt.Errorf("ytdlp: create job object: %w", err)
	}

	r, w, err := os.Pipe()
	if err != nil {
		_ = windows.CloseHandle(job)
		return nil, fmt.Errorf("ytdlp: create pipe: %w", err)
	}

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = w
	cmd.Stderr = w
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}

	if err := cmd.Start(); err != nil {
		_ = r.Close()
		_ = w.Close()
		_ = windows.CloseHandle(job)
		return nil, fmt.Errorf("ytdlp: start %s: %w", name, err)
	}

	// The parent must drop its copy of the write end so the read side sees EOF
	// when the child exits.
	_ = w.Close()

	// Assign the process to the job so its children are captured too. yt-dlp
	// spawns ffmpeg only at the merge step (seconds later), so there is no race
	// in practice; a failure here degrades to "no job protection", not an error.
	procHandle, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		_ = r.Close()
		_ = windows.CloseHandle(job)
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("ytdlp: open process: %w", err)
	}
	assignErr := windows.AssignProcessToJobObject(job, procHandle)
	_ = windows.CloseHandle(procHandle)
	if assignErr != nil {
		// Not fatal: the download still works, cancel just falls back to killing
		// the direct child only.
		assignErr = nil
	}

	return &winChild{
		cmd: cmd,
		job: job,
		r:   r,
		w:   w,
	}, nil
}

// createKillOnCloseJob creates a Job Object whose processes are terminated when
// the last handle to the job is closed (JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE).
func createKillOnCloseJob() (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}

	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	_, err = windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	)
	if err != nil {
		_ = windows.CloseHandle(job)
		return 0, err
	}
	return job, nil
}

func (c *winChild) stdout() io.ReadCloser { return c.r }

func (c *winChild) wait() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.waited {
		return c.err
	}
	c.waited = true
	c.err = c.cmd.Wait()
	return c.err
}

// jobProcessIDs lists the PIDs currently assigned to the job. yt-dlp's onefile
// build is a PyInstaller bootloader that re-executes itself as a python.exe
// child, so pausing only the direct child would leave the real downloader
// running; we must suspend every process in the job.
func jobProcessIDs(job windows.Handle) ([]uint32, error) {
	var info jobBasicProcessIDList
	err := windows.QueryInformationJobObject(
		job,
		windows.JobObjectBasicProcessIdList,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
		nil,
	)
	if err != nil {
		return nil, err
	}
	n := int(info.NumberOfProcessIDsInList)
	if n > len(info.ProcessIDList) {
		n = len(info.ProcessIDList)
	}
	pids := make([]uint32, 0, n)
	for i := 0; i < n; i++ {
		pids = append(pids, uint32(info.ProcessIDList[i]))
	}
	return pids, nil
}

// forEachJobProcess opens each process in the job and applies fn (suspend or
// resume). Processes that have already exited are skipped.
func (c *winChild) forEachJobProcess(fn func(handle windows.Handle) error) error {
	if c.job == 0 {
		return errors.New("ytdlp: no job object")
	}
	pids, err := jobProcessIDs(c.job)
	if err != nil {
		return err
	}
	var firstErr error
	for _, pid := range pids {
		h, oerr := windows.OpenProcess(windows.PROCESS_SUSPEND_RESUME, false, pid)
		if oerr != nil {
			continue // process already gone
		}
		if ferr := fn(h); ferr != nil && firstErr == nil {
			firstErr = ferr
		}
		_ = windows.CloseHandle(h)
	}
	return firstErr
}

// suspend pauses the whole process tree via NtSuspendProcess.
func (c *winChild) suspend() error {
	if c.job == 0 {
		return errors.New("ytdlp: no job object")
	}
	return c.forEachJobProcess(ntSuspend)
}

// resume continues a suspended process tree.
func (c *winChild) resume() error {
	if c.job == 0 {
		return errors.New("ytdlp: no job object")
	}
	return c.forEachJobProcess(ntResume)
}

// kill terminates the whole job (yt-dlp plus any ffmpeg child), then closes the
// job handle so no further children survive.
func (c *winChild) kill() error {
	var err error
	if c.job != 0 {
		if terr := windows.TerminateJobObject(c.job, 1); terr != nil {
			err = terr
		}
	}
	if c.cmd.Process != nil {
		if perr := c.cmd.Process.Kill(); perr != nil && err == nil {
			err = perr
		}
	}
	return err
}

// close releases the pipe and job handles. Closing the job terminates any
// stragglers (KILL_ON_JOB_CLOSE), guaranteeing no orphan processes.
func (c *winChild) close() {
	if c.r != nil {
		_ = c.r.Close()
	}
	if c.job != 0 {
		_ = windows.CloseHandle(c.job)
		c.job = 0
	}
}

// ntSuspend/ntResume call the ntdll routines and translate the NTSTATUS code.
func ntSuspend(h windows.Handle) error {
	ret, _, _ := procNtSuspendProcess.Call(uintptr(h))
	if ret != 0 {
		return fmt.Errorf("ytdlp: NtSuspendProcess failed (status 0x%x)", ret)
	}
	return nil
}

func ntResume(h windows.Handle) error {
	ret, _, _ := procNtResumeProcess.Call(uintptr(h))
	if ret != 0 {
		return fmt.Errorf("ytdlp: NtResumeProcess failed (status 0x%x)", ret)
	}
	return nil
}
