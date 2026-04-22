#!/usr/bin/env bash

resolve_secret_input() {
  local name="$1"
  local file_name="${name}_FILE"
  local current="${!name-}"
  local file_path="${!file_name-}"

  if [[ -n "${current}" && -n "${file_path}" ]]; then
    printf 'error: %s and %s are both set; set only one\n' "${name}" "${file_name}" >&2
    return 1
  fi

  if [[ -z "${file_path}" ]]; then
    return 0
  fi

  if [[ ! -r "${file_path}" ]]; then
    printf 'error: %s points to an unreadable file: %s\n' "${file_name}" "${file_path}" >&2
    return 1
  fi

  local value
  value="$(<"${file_path}")"
  if [[ -z "${value}" ]]; then
    printf 'error: %s resolved to an empty value\n' "${file_name}" >&2
    return 1
  fi

  printf -v "${name}" '%s' "${value}"
  export "${name}"
}
