# MyGo experiment evidence — 6 October 2026

## Inputs

- Widget base: `41beef6` (repository `tcballard/omarchy-widget-github`).
- MyGo: v0.2.11, `eac977aff4f2b4d50b76ac98b5203efb7630519a`.
- Core parser reference: `70712a403d6513893c0e4c6c0e9e177dfba646da`;
  also inspected current Core `e1b8c1582d0a39cafe9a46add383b7651c6fa649`.
- Runtime: Linux x86_64, Go 1.27.1. Native UI tests use MyGo's headless tester;
  no XPS or Hyprland session is available here.
- `SOURCE_SHA256SUMS` identifies implementation, dependency lock and test inputs.
  It records files, not proof of execution. CI artifacts separately record the
  full commit and executable checksum.

## Reproduced now

- `GITHUB_CALENDAR_FIXTURE=/tmp/widgets-tcballard-contributions.html go test -race ./...`:
  passed. Includes real captured fragment, 365/366-day calendars, Sunday-padded
  365–371-day calendars, truncated/oversized/duplicate/gapped/invalid data, zero
  activity, user validation, mocked HTTP failures/redirects/size/cancellation,
  stale state, superseded generations, size/palette controls and keyboard input.
- `go vet ./...`: passed.
- `go mod verify`: all modules verified.
- `CGO_ENABLED=0 go build -mod=readonly -tags mygo_noinspector -trimpath
  -ldflags='-s -w -X github.com/egoist/mygo.production=1' -o build/github-widget-mygo .`:
  passed.
- `build/github-widget-mygo --check --username tcballard`: passed through the
  actual Go HTTPS transport: **6,839 public contributions, 367 days,
  2025-10-05 through 2026-10-06**. This number is a point-in-time observation.
- Captured curl response: 242,158 bytes, SHA-256
  `13b8e853e86d79b10074f21abee85b0468fb190d27f700b364acb34e195bf580`.
  Its earlier count was 6,838; the parser test passed against those bytes. The
  capture is a temporary test input, not bundled user data.
- Native offscreen `--demo --render` at all three sizes: generated and inspected.
  Bundled previews display synthetic data and are not live desktop captures.
- Existing `node tests/model.cjs`: passed; existing QML sources/manifest unchanged.

## Failed first, then corrected

The first captured-fragment test and live `--check` failed because the inherited
Core parser accepted only 365–366 days. The 367-day response starts on Sunday.
The experiment now accepts 365–371 contiguous days, requiring Sunday alignment
above 366; regression tests cover the full bound and reject 372 days or a wrongly
aligned padded year. This does not change the Core repository or installed widget.

Installing Xvfb here failed because apt could not change its sandbox user/group.
No native-window launch is claimed. Headless UI rendering/tests work without it.

## Not run / still required

- Live Omarchy/Hyprland: window identity, focus, keyboard close, placement,
  workspace behaviour, display scaling/hotplug, suspend/resume and theme switching.
- Real offline transition and canceled request shutdown on the XPS. Portable
  transport/state tests are narrower than these desktop scenarios.
- XPS CPU/RSS, comparison against Core's process tree, or GPU power consumption.
- Core's existing Rust/QML package-integration suite locally (no local Rust/Qt
  development toolchain); the repository's existing CI remains responsible for it.
- No desktop layer-shell hosting, Core permissions/sandbox or live widget embedding
  is implemented. This is a standalone rendering experiment.

Options are ephemeral, network access is public-profile only, and removal requires
closing the window and deleting the fresh source/binary directory. No system files,
installed widgets, Core registry entries or settings are changed.
