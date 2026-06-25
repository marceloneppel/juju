run_relation_default_endpoint() {
	echo

	model_name="test-relation-default-endpoint"
	file="${TEST_DIR}/${model_name}.log"

	ensure "${model_name}" "${file}"

	# certs-provider provides a single "certs" endpoint on the "shared" interface.
	# db-app requires two endpoints on the same interface, certs-main (default)
	# and certs-extra, so the shorthand `juju integrate` is ambiguous.
	echo "Deploy a provider with one shared endpoint and a requirer with two (certs-main default)"
	# shellcheck disable=SC2046
	juju deploy $(pack_charm ./testcharms/charms/relation-default-provider) certs-provider
	# shellcheck disable=SC2046
	juju deploy $(pack_charm ./testcharms/charms/relation-default-requirer) db-app

	wait_for "certs-provider" "$(idle_condition "certs-provider")"
	wait_for "db-app" "$(idle_condition "db-app")"

	# Without the default flag this shorthand is ambiguous: certs <-> certs-main and
	# certs <-> certs-extra both match. The default flag on certs-main must resolve
	# the shorthand to certs-provider:certs <-> db-app:certs-main (juju #19615).
	echo "Shorthand integrate resolves to the default endpoint (certs-main)"
	juju integrate certs-provider db-app
	wait_for "certs-main" '.applications."db-app".relations | keys | .[]'

	# The shorthand must have chosen certs-main, not certs-extra.
	endpoints=$(juju status --format json | yq -r '.applications."db-app".relations | keys | .[]')
	check_contains "${endpoints}" "certs-main"
	if echo "${endpoints}" | grep -qx "certs-extra"; then
		# shellcheck disable=SC2046
		echo $(red "shorthand integrate unexpectedly used certs-extra")
		exit 1
	fi

	# Explicit endpoint selection is unaffected by the feature (backward compatibility):
	# the previously-working explicit form still creates a certs <-> certs-extra relation.
	echo "Explicit endpoint selection still works"
	juju integrate certs-provider:certs db-app:certs-extra
	wait_for "certs-extra" '.applications."db-app".relations | keys | .[]'

	# Meta.Check rejects a charm with more than one default endpoint in a
	# (role, interface) set, since that provides no disambiguation.
	echo "A charm with more than one default endpoint is rejected at deploy time"
	# shellcheck disable=SC2046
	got=$(juju deploy $(pack_charm ./testcharms/charms/relation-default-invalid) invalid 2>&1 || true)
	check_contains "${got}" "more than one requires endpoint marked as default"

	destroy_model "${model_name}"
}

test_relation_default_endpoint() {
	if [ "$(skip 'test_relation_default_endpoint')" ]; then
		echo "==> TEST SKIPPED: relation default endpoint tests"
		return
	fi

	(
		set_verbosity

		cd .. || exit

		run "run_relation_default_endpoint"
	)
}
