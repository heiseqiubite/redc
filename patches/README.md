# patches/

This directory contains local patches for upstream Go modules.

## terraform-exec (github.com/hashicorp/terraform-exec v0.24.0)

**Problem**: On Windows, when redc-gui.exe (built with `-H windowsgui`, no console) spawns
terraform subprocesses via terraform-exec, Windows allocates a new visible console window for
each terraform command (init, plan, apply, destroy), causing black command windows to flash
on screen.

**Root cause**: `tfexec/cmd_default.go` (build tag: `!linux`) sets no `SysProcAttr`, so on
Windows the child process inherits no console from the GUI parent and Windows creates a new
visible one. The fix requires `SysProcAttr.HideWindow = true` at the process creation level.

**Upstream issue**: https://github.com/hashicorp/terraform-exec/issues/570

**Patch**: Added `tfexec/cmd_windows.go` with `SysProcAttr{HideWindow: true}`.
Changed `cmd_default.go` build tag from `!linux` → `!linux && !windows` to avoid conflict.

**To remove this patch** (when upstream fixes it):
1. Remove `replace github.com/hashicorp/terraform-exec => ./patches/terraform-exec` from go.mod
2. Delete this patches/terraform-exec directory
