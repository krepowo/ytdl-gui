# Spec: settings-screen

## Objective
Let the user configure download directory, max concurrent downloads, the default
quality/format, and browser cookies for gated sites, reusing the same icon-first
visual language as the main screen.

## Tech Stack
React 19 + TypeScript, MUI v9 form controls, Tailwind v4 layout.

## Commands
- Test: `npm run test`
- Typecheck: `npm run typecheck`

## Project Structure
```
frontend/src/features/settings/
  SettingsScreen.tsx   → form bound to GetSettings/SaveSettings
  DirPicker.tsx        → read-only field + folder icon button (PickDownloadDir)
  ConfigPathNote.tsx   → shows the active config path (install dir or fallback)
```

## Code Style
```tsx
// Save on change with debounce; toast on success.
const onSave = async (next: Settings) => {
  await SaveSettings(next);
  setSaved(true);
};
```

## Testing Strategy
RTL: loading settings populates the form; changing concurrency calls
`SaveSettings` with the new value; the folder icon button fills the dir from
`PickDownloadDir`; the config-path note renders the value from `GetAppInfo`.

## Boundaries
- **Always:** clamp concurrency to 1–8 in the UI; show the resolved dir path and
  the active config-file path.
- **Ask first:** adding new settings fields.
- **Never:** let an invalid value reach `SaveSettings`.

## Success Criteria
- [ ] Settings load on open and persist after save + app restart.
- [ ] Concurrency control limited to 1–8.
- [ ] Folder icon opens the native dialog and updates the field.
- [ ] The active config path is displayed (install dir, or the `%APPDATA%` fallback).

## Open Questions
- None.
