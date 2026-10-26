# stcopenmpp Ubuntu Install Scripts

Shell scripts for installing [StatCan's OpenM++ distribution (stcopenmpp)](https://github.com/StatCan/stcopenmpp) on Ubuntu, along with the compiler toolchain and build tools it needs.

| Script | Purpose | Needs `sudo` | Installs to |
|---|---|---|---|
| `install_ompp_ubuntu_dependencies.sh` | Installs GCC/G++ 12 and the build tools *stcopenmpp needs* | Yes | System packages |
| `install_stcompp_ubuntu.sh` | System-wide install for all users | Yes | `/opt` |
| `install_stcompp_ubuntu_user.sh` | Single-user install, no admin rights needed | No | `$HOME` |

Run the dependencies script first, then **one** of the two install scripts.

---

## Requirements

- Ubuntu. 22.04 gets the `ubuntu22` build; every other release gets the generic `ubuntu` build.
- Internet access to `github.com`.
- `git`, `wget`, `tar` and `lsb_release` (present on a standard Ubuntu install).
- `vim` (system-wide script only; it is used to parse the list of release tags).
- An interactive terminal: both install scripts prompt for input.

---

## 1. `install_ompp_ubuntu_dependencies.sh`

This script sets up the compiler toolchain and build utilities.

```bash
bash install_ompp_ubuntu_dependencies.sh
```

It does the following:

1. Installs `gcc-12` and `g++-12`.
2. **Removes all existing `update-alternatives` entries for `gcc` and `g++`**, then registers:
   - `gcc-11` (priority 10) and `gcc-12` (priority 20), which makes **GCC 12 the default**
   - `g++-11` (priority 10) and `g++-12` (priority 20), which makes **G++ 12 the default**
   - `cc` → `gcc` and `c++` → `g++`
3. Installs `sqlite`, `gdb`, `lynx`, `flex`, `bison` and `make`.

> ⚠️ It assumes `gcc-11` / `g++-11` already exist, as they do by default on Ubuntu 22.04. On other releases those two alternative entries will point at missing binaries.

---

## 2. `install_stcompp_ubuntu.sh` (system-wide)

This script installs **stcopenmpp** under `/opt` and configures it for every user on the machine.

```bash
bash install_stcompp_ubuntu.sh
```

It does the following:

1. **Pick a version.** It installs `lynx` and `dos2unix`, scrapes the release tags from `https://github.com/statcan/stcopenmpp/tags`, and lists them (newest first). You type the number of the version you want and confirm with `y`.
2. **Build the download URL** from your Ubuntu release:
   ```
   https://github.com/StatCan/stcopenmpp/releases/download/<version>/stcopenmpp-ubuntu[22]-<version>-gh-release.tar.xz
   ```
3. **Download and extract** the archive into `/opt/stcopenmpp-ubuntu[22]-<version>`, then delete the archive.
4. **Build an ASAN copy.** It copies the install to `/opt/<install>_ASAN` and rebuilds `libopenm` there with AddressSanitizer (`USE_ASAN=1`), in both debug and release (`RELEASE=1`) variants. The ASAN copy is then made world-writable (`chmod -R 777`).
5. **Update `/etc/bash.bashrc`** (system-wide shell config):
   - backs it up to `~/bash.bashrc.bak`
   - removes any earlier OpenM++ lines (`OMP_ROOT`, `OM_ROOT`, `OMA_ROOT`, `LD_LIBRARY_PATH`, the stack `ulimit`)
   - appends:
     ```bash
     ulimit -S -s 65536
     export OMP_ROOT=/opt/<install>
     export OM_ROOT=/opt/<install>
     export OMA_ROOT=/opt/<install>_ASAN
     ```
6. **Update `~/.bashrc`** for the user running the script, with the same lines but `OMA_ROOT=` left empty.

The script pauses with *"Press the [ENTER] key to continue"* between stages.

---

## 3. `install_stcompp_ubuntu_user.sh` (single user)

This script installs **stcopenmpp** in your home directory and needs no `sudo`.

```bash
bash install_stcompp_ubuntu_user.sh
```

It does the following:

1. **Pick a version.** It clones (or `git pull`s) `https://github.com/StatCan/stcopenmpp` into `~/stcopenmpp`, lists the repository's `git tag`s, and asks you to choose one by number and confirm. The clone is deleted afterwards.
2. **Build the download URL** the same way as the system-wide script (`ubuntu22` build on 22.04, generic `ubuntu` build otherwise).
3. **Download and extract** into `~/stcopenmpp-ubuntu[22]-<version>`.
4. **Build an ASAN copy** at `~/<install>_ASAN`, rebuilding `libopenm` in `openm/` with `USE_ASAN=1`, in debug and release variants.
5. **Update `~/.bashrc`.** It backs the file up to `~/.bashrc.bak` (or creates it if missing), removes earlier OpenM++ lines, and appends:
   ```bash
   ulimit -S -s 65536
   export OMP_ROOT=$HOME/<install>
   export OM_ROOT=$HOME/<install>
   export OMA_ROOT=$HOME/<install>_ASAN
   ```

---

## Environment variables set

| Variable | Meaning |
|---|---|
| `OM_ROOT` | Root of the **stcopenmpp** installation (used by OpenM++ makefiles and tools) |
| `OMP_ROOT` | Same path as `OM_ROOT` |
| `OMA_ROOT` | Root of the AddressSanitizer build, for memory-error debugging |
| `ulimit -S -s 65536` | Raises the soft stack limit to 64 MB, which large models need |

**After installing, open a new terminal** (or run `source ~/.bashrc`). The `source` calls inside the scripts only affect the script's own shell, not yours.

To check it worked:

```bash
echo $OM_ROOT
```

---

## Uninstalling / rolling back

- Delete the install folders: `/opt/stcopenmpp-*` (system-wide) or `~/stcopenmpp-*` (user).
- Restore your shell config from the backups:
  - `~/.bashrc.bak` → `~/.bashrc`
  - `~/bash.bashrc.bak` → `/etc/bash.bashrc` (needs `sudo`)
- The compiler defaults changed by the dependencies script can be reviewed with `update-alternatives --display gcc`.

