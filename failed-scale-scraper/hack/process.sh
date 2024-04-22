#!/opt/homebrew/bin/bash

# This script takes a json array of log files and counts the number of events of each type. It requires bash 4+.

inputFile="$1"

function getCount() {
    message=$1
    echo "$(yq "[.[] | select (.message == \"${message}\")] | length" -oy $inputFile)"
}

declare -A events
events['created-nodeclaim']="created nodeclaim"
events['launched-nodeclaim']="launched nodeclaim"
events['registered-nodeclaim']="registered nodeclaim"
events['initialized-nodeclaim']="initialized nodeclaim"
events['deleted-nodeclaim']="deleted nodeclaim"
events['tainted-node']="tainted node"
events['deleted-node']="deleted node"
events['interruption-message']="initiating delete from interruption message"

for event in ${!events[@]}; do
    query=${events[${event}]}
    printf "%s: %d\n" ${event} $(getCount "${query}")
done

