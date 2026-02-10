PODMAN="./bin/podman"

test_quadlet_volume() {
  local KEYWORD="$1"
  NAME=$KEYWORD
  local JSON_KEY="$2"
  local VALUE="$3"

  local quadlet_dir="${HOME}/.config/containers/systemd"
  local generator_dir="/run/user/$(id -u)/systemd/generator"
  local quadlet_file="${quadlet_dir}/${NAME}.volume"
  local service_file="${generator_dir}/${NAME}-volume.service"

  mkdir -p "$quadlet_dir"
  rm -f "$quadlet_file"

  cat <<EOF > "$quadlet_file"
[Volume]
VolumeName=${NAME}
${KEYWORD}=${VALUE}
EOF

  systemctl --user daemon-reload

  [[ -f "$service_file" ]] && echo "ok" || echo "not ok"
  grep -q -- "$KEYWORD" "$service_file" && echo "ok" || echo "not ok"

  systemctl --user start "${NAME}-volume.service"

  "$PODMAN" volume inspect --format "{{.${JSON_KEY}}}" "$NAME" | grep -q -- "$VALUE" && echo "ok" || echo "not ok"
}

test_quadlet_volume "VolumeUID" "UID" "1234"
test_quadlet_volume "VolumeGID" "GID" "5678"
