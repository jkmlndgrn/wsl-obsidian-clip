#!/bin/sh
set -eu

install_dir=${1:?install directory is required}
tool_name=${2:-wsl-obsidian-clip}
home_dir=${HOME:?HOME is required}

case ":${PATH:-}:" in
*":$install_dir:"*)
  printf '%s is already on PATH.\n' "$install_dir"
  exit 0
  ;;
esac

contains_path_config() {
  file=$1
  dir=$2

  if grep -F "$dir" "$file" >/dev/null 2>&1; then
    return 0
  fi
  if [ "$dir" = "$home_dir/.local/bin" ]; then
    grep -F "$HOME/.local/bin" "$file" >/dev/null 2>&1 && return 0
  fi
  return 1
}

shell_name=${SHELL:-}
shell_name=${shell_name##*/}
profile=
profile_type=posix

case "$shell_name" in
fish)
  profile="$home_dir/.config/fish/config.fish"
  profile_type=fish
  ;;
zsh)
  profile="$home_dir/.zshrc"
  ;;
bash)
  profile="$home_dir/.bashrc"
  ;;
sh | dash)
  profile="$home_dir/.profile"
  ;;
*)
  for candidate in "$home_dir/.profile" "$home_dir/.bashrc" "$home_dir/.zshrc"; do
    if [ -f "$candidate" ]; then
      profile=$candidate
      break
    fi
  done
  if [ -z "$profile" ]; then
    profile="$home_dir/.profile"
  fi
  ;;
esac

mkdir -p "$(dirname "$profile")"

if [ -f "$profile" ] && contains_path_config "$profile" "$install_dir"; then
  printf 'PATH update skipped: %s already appears in %s.\n' "$install_dir" "$profile"
  printf 'Open a new shell if %s is still not available.\n' "$tool_name"
  exit 0
fi

path_expr=$install_dir
if [ "$install_dir" = "$home_dir/.local/bin" ]; then
  path_expr="$HOME/.local/bin"
fi

if [ "$profile_type" = fish ]; then
  printf '\n# Added by %s installer\nfish_add_path -g "%s"\n' "$tool_name" "$path_expr" >>"$profile"
  printf 'Added %s to PATH in %s.\n' "$install_dir" "$profile"
  printf "Open a new shell or run: source '%s'\n" "$profile"
else
  printf "\n# Added by %s installer\ncase \":$PATH:\" in\n  *\":%s:\"*) ;;\n  *) export PATH=\"%s:$PATH\" ;;\nesac\n" "$tool_name" "$path_expr" "$path_expr" >>"$profile"
  printf 'Added %s to PATH in %s.\n' "$install_dir" "$profile"
  printf "Open a new shell or run: . '%s'\n" "$profile"
fi
