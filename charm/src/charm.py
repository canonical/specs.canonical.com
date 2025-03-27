#!/usr/bin/env python3
# Copyright 2024 Goulin
# See LICENSE file for licensing details.

"""Go Charm entrypoint."""

import logging
import typing

import ops
import paas_charm.go
from charms.nginx_ingress_integrator.v0.nginx_route import require_nginx_route

logger = logging.getLogger(__name__)


class SpecsCharm(paas_charm.go.Charm):
    """Go Charm service."""

    def __init__(self, *args: typing.Any) -> None:
        """Initialize the instance.

        Args:
            args: passthrough to CharmBase.
        """
        super().__init__(*args)
        require_nginx_route(
            charm=self,
            service_hostname=self.app.name,
            service_name=self.app.name,
            service_port=self._workload_config.port
        )

    def _on_ingress_ready(self, _: ops.HookEvent) -> None:
        return

    def _on_ingress_address_changed(self, _: ops.Event) -> None:
        return


if __name__ == "__main__":
    ops.main(SpecsCharm)
