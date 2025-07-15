#!/bin/sh

# Define functions.
function fixperms {
	# /opt/whatsapp-cloud is read-only, so disable file logging if it's pointing there.
	if [[ "$(yq e '.logging.writers[1].filename' /data/config.yaml)" == "./logs/whatsapp-cloud.log" ]]; then
		yq -I4 e -i 'del(.logging.writers[1])' /data/config.yaml
	fi
}

if [[ ! -f /data/config.yaml ]]; then
	/usr/bin/whatsapp-cloud -c /data/config.yaml -e
	echo "Didn't find a config file."
	echo "Copied default config file to /data/config.yaml"
	echo "Modify that config file to your liking."
	echo "Start the container again after that to generate the registration file."
	exit
fi

if [[ ! -f /data/registration.yaml ]]; then
	/usr/bin/whatsapp-cloud -g -c /data/config.yaml -r /data/registration.yaml || exit $?
	echo "Didn't find a registration file."
	echo "Generated one for you."
	echo "See https://docs.mau.fi/bridges/general/registering-appservices.html on how to use it."
	exit
fi

cd /data
fixperms

if [[ "${1}" == "dev" ]]; then
	/usr/bin/whatsapp-cloud -c /data/config.yaml
fi

exec /usr/bin/whatsapp-cloud
