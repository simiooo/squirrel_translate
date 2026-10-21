# Project instructions

This is a Windows desktop translation prototype built in Go.

Use Nuto staging for project tracking. The user chose the existing Goood
workspace (`simooowann`), rather than creating a new workspace. Always pass
`--env staging --workspace simooowann --project squirrel_translatre`.
Never store API keys in the repository, issue notes, or logs.

Keep native capture and window placement separate from model and vocabulary
logic. Closing an unpinned translation must cancel its request and release
session resources. Pinned translations retain their own immutable selection.

Release packages must be built by GitHub Actions. Push a version tag to run
the Windows release workflow and create a draft GitHub Release; do not upload
locally built binaries as official release assets.

Keep README.md and README.en.md focused on users, with equivalent Chinese
and English installation and usage instructions. Cover both GitHub Releases
and PowerShell 5.1 installation; keep implementation details in docs/.
