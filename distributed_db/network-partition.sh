#!/usr/bin/env bash
# network-partition.sh — simulate a network partition against a Cassandra
# node, without stopping its process (unlike `docker stop`).
#
# `docker stop` kills the JVM outright: gossip immediately marks the node as
# down and hints start accumulating. A real network partition is different —
# the process keeps running, keeps its in-memory state, and from its own
# point of view the *rest* of the cluster looks unreachable. This script uses
# `docker network disconnect/connect` to sever/restore a container's network
# attachment, which is a much closer approximation of that failure mode.
#
# Usage:
#   ./network-partition.sh isolate <container>   # cut the node off from the cluster network
#   ./network-partition.sh heal <container>      # reconnect the node to the cluster network
#   ./network-partition.sh status                # show which nodes are currently attached

set -euo pipefail

usage() {
	echo "Usage: $0 {isolate|heal} <container-name>" >&2
	echo "       $0 status" >&2
	echo "Example: $0 isolate cassandra-3" >&2
	exit 1
}

[[ $# -ge 1 ]] || usage

action="$1"

# Discover the docker-compose network name dynamically instead of hardcoding
# it, since the project directory name affects the generated network name.
network_for() {
	docker inspect -f '{{range $k, $v := .NetworkSettings.Networks}}{{$k}}{{end}}' "$1"
}

case "$action" in
isolate)
	[[ $# -eq 2 ]] || usage
	container="$2"
	network="$(network_for "$container")"
	if [[ -z "$network" ]]; then
		echo "Could not determine network for $container (is it running?)" >&2
		exit 1
	fi
	echo "Disconnecting $container from network '$network' (process keeps running)..."
	docker network disconnect "$network" "$container"
	echo "$container is now partitioned away from the cluster."
	echo "Watch the effect with: docker exec cassandra-1 nodetool status"
	;;
heal)
	[[ $# -eq 2 ]] || usage
	container="$2"
	network="$(docker inspect -f '{{range $k, $v := .NetworkSettings.Networks}}{{$k}}{{end}}' cassandra-1)"
	if [[ -z "$network" ]]; then
		echo "Could not determine cluster network from cassandra-1 (is it running?)" >&2
		exit 1
	fi
	echo "Reconnecting $container to network '$network'..."
	docker network connect "$network" "$container"
	echo "$container has rejoined the cluster network."
	echo "Watch it rejoin with: docker exec cassandra-1 nodetool status"
	;;
status)
	for c in cassandra-1 cassandra-2 cassandra-3 cassandra-4; do
		if docker inspect "$c" >/dev/null 2>&1; then
			net="$(network_for "$c")"
			if [[ -n "$net" ]]; then
				echo "$c: attached to '$net'"
			else
				echo "$c: NOT attached to any network (partitioned)"
			fi
		fi
	done
	;;
*)
	usage
	;;
esac
