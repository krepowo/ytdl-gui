export namespace app {
	
	export class AppInfo {
	    version: string;
	    configPath: string;
	    ytDlpVersion: string;
	    ffmpegVersion: string;
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.configPath = source["configPath"];
	        this.ytDlpVersion = source["ytDlpVersion"];
	        this.ffmpegVersion = source["ffmpegVersion"];
	    }
	}
	export class DownloadRequest {
	    url: string;
	    mode: string;
	    formatId: string;
	    needsMerge: boolean;
	    title: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.mode = source["mode"];
	        this.formatId = source["formatId"];
	        this.needsMerge = source["needsMerge"];
	        this.title = source["title"];
	    }
	}

}

export namespace queue {
	
	export class Job {
	    id: string;
	    url: string;
	    title: string;
	    mode: string;
	    formatId: string;
	    state: string;
	    percent: number;
	    downloadedBytes: number;
	    totalBytes: number;
	    speedBps: number;
	    etaSec: number;
	    outputPath: string;
	    error: string;
	    createdAt: number;
	
	    static createFrom(source: any = {}) {
	        return new Job(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.url = source["url"];
	        this.title = source["title"];
	        this.mode = source["mode"];
	        this.formatId = source["formatId"];
	        this.state = source["state"];
	        this.percent = source["percent"];
	        this.downloadedBytes = source["downloadedBytes"];
	        this.totalBytes = source["totalBytes"];
	        this.speedBps = source["speedBps"];
	        this.etaSec = source["etaSec"];
	        this.outputPath = source["outputPath"];
	        this.error = source["error"];
	        this.createdAt = source["createdAt"];
	    }
	}

}

export namespace settings {
	
	export class Settings {
	    downloadDir: string;
	    maxConcurrent: number;
	    defaultMode: string;
	    defaultQuality: string;
	    filenameTemplate: string;
	    cookiesBrowser: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.downloadDir = source["downloadDir"];
	        this.maxConcurrent = source["maxConcurrent"];
	        this.defaultMode = source["defaultMode"];
	        this.defaultQuality = source["defaultQuality"];
	        this.filenameTemplate = source["filenameTemplate"];
	        this.cookiesBrowser = source["cookiesBrowser"];
	    }
	}

}

export namespace ytdlp {
	
	export class FormatOption {
	    formatId: string;
	    label: string;
	    ext: string;
	    height: number;
	    width: number;
	    fps: number;
	    bitrate: number;
	    filesize: number;
	    needsMerge: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FormatOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.formatId = source["formatId"];
	        this.label = source["label"];
	        this.ext = source["ext"];
	        this.height = source["height"];
	        this.width = source["width"];
	        this.fps = source["fps"];
	        this.bitrate = source["bitrate"];
	        this.filesize = source["filesize"];
	        this.needsMerge = source["needsMerge"];
	    }
	}
	export class MediaInfo {
	    id: string;
	    title: string;
	    uploader: string;
	    duration: number;
	    thumbnail: string;
	    extractor: string;
	    webpageUrl: string;
	    isLive: boolean;
	    videoOptions: FormatOption[];
	    audioOptions: FormatOption[];
	
	    static createFrom(source: any = {}) {
	        return new MediaInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.uploader = source["uploader"];
	        this.duration = source["duration"];
	        this.thumbnail = source["thumbnail"];
	        this.extractor = source["extractor"];
	        this.webpageUrl = source["webpageUrl"];
	        this.isLive = source["isLive"];
	        this.videoOptions = this.convertValues(source["videoOptions"], FormatOption);
	        this.audioOptions = this.convertValues(source["audioOptions"], FormatOption);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

