# scripts/

Office git attribution — [TheBoringMajdoor](https://github.com/themajdoor).

Every commit through the office can carry:

```text
Co-authored-by: TheBoringMajdoor <themajdoor@theboring.name>
```

On by default (`"attribution": "on"` in `brain.json`). Office auto-installs a `commit-msg` hook in the repo it boots in.

For a repo the office never opens:

```bash
./scripts/install-majdoor-hook.sh /path/to/repo
# or
curl -fsSL \
  https://raw.githubusercontent.com/theboringhumane/theboringoffice/main/scripts/install-majdoor-hook.sh | sh -s -- /path/to/repo
```

`--uninstall` peels the hook and restores the backup. `install.sh --majdoor-hook /path/to/repo` does the same.

| File | |
|---|---|
| `install-majdoor-hook.sh` | install / uninstall |
| `majdoor-commit-msg-hook.sh` | hook body |
| `majdoor-env.sh` | `GIT_AUTHOR_*` / `GIT_COMMITTER_*` when `THEBORINGOFFICE_AUTO_COMMIT=true` |

Office auto-commits (`THEBORINGOFFICE_AUTO_COMMIT=true`) are *authored* by the majdoor. Hand-written commits keep your identity and pick up the trailer via the hook. Git hooks cannot set author env for the parent process — that layer is env, not a hook.

`"attribution": "off"` in `brain.json` turns it off and removes the office-installed hook only (byte-identical; foreign hooks stay).
