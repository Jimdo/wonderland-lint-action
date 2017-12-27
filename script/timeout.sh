#!/bin/sh

TIMEOUT_SECONDS=$1
TIMEOUT_EXITCODE=$2

MIN_EXITCODE=1
MAX_EXITCODE=253

trap 'exit' SIGTERM

if [ -z "$TIMEOUT_SECONDS" -o -z "$TIMEOUT_EXITCODE" ]; then
    echo "ERROR: param missing"
    echo "Syntax: $(basename $0) <timeout_seconds> <timeout_exitcode>"
    exit 254
fi

for param in "$TIMEOUT_SECONDS" "$TIMEOUT_EXITCODE"; do
	if ! echo $param | egrep '^[0-9]+$' > /dev/null 2>&1; then
		echo "$param is not a number"
		exit 254
	fi
done

if [ "$TIMEOUT_EXITCODE" -lt "$MIN_EXITCODE" -o "$TIMEOUT_EXITCODE" -gt "$MAX_EXITCODE" ];then
    echo "ERROR: timeout exit code has to be from ${MIN_EXITCODE} to ${MAX_EXITCODE}"
    exit 254
fi

sleep "$TIMEOUT_SECONDS" &
wait $!

exit "$TIMEOUT_EXITCODE"
