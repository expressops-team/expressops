#!/bin/bash

set -e

GREEN="\033[32m"
RED="\033[31m"
BLUE="\033[34m"
RESET="\033[0m"


PLUGINS=(
  "plugins/healthcheck/health_check.go"
  #"plugins/kubehealth/kube_health.go"
  "plugins/sleep/sleep_plugin.go"
  "plugins/slack/slack.go"
  "plugins/testprint/testprint.go"
  "plugins/clean-disk/clean_disk.go"
  "plugins/logfilecreator/logfilecreator.go"
  #"plugins/logcleaner/logcleaner.go"
)

for plugin in "${PLUGINS[@]}"; do
  plugin_dir=$(dirname "$plugin")
  plugin_file=$(basename "$plugin")
  output_file="${plugin_dir}/$(basename "${plugin_file%.go}.so")"

  echo -e "${BLUE}Compilando  ${RED}${plugin_file}...${RESET}"
  go build -buildmode=plugin -o "$output_file" "$plugin" && \
    echo -e "${GREEN}✅ ${plugin_file} listo en ${output_file}${RESET}"
done

echo -e "${GREEN}TODO LISTO${RESET}"
echo -e "${YELLOW}🎉 Ahora a ejecutarlo${RESET}"
go run cmd/expressops.go 

# .plugin_builder/script.sh