# Runtime Package Layout

Each runtime package follows:

- `script/runtime/<runtime>/<version>/manifest.json`
- `script/runtime/<runtime>/<version>/runtime-package.*`
- `script/runtime/<runtime>/<version>/preflight.sh`
- `script/runtime/<runtime>/<version>/install.sh`
- `script/runtime/<runtime>/<version>/verify.sh`
- `script/runtime/<runtime>/<version>/uninstall.sh`

`manifest.json` fields:

- `runtime`: runtime identifier (`k8s` or `compose`)
- `version`: package version
- `package_file`: binary/package artifact name
- `sha256`: package checksum
- `preflight_script`/`install_script`/`verify_script`/`uninstall_script`: relative script paths

The backend validates `manifest.json`, verifies package checksum, then executes phase scripts on remote hosts via SSH.
