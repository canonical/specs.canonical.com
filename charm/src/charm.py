#!/usr/bin/env python3
# Copyright 2024 Goulin
# See LICENSE file for licensing details.

"""Go Charm entrypoint."""

import logging
import typing

import ops

import paas_charm.go
from paas_charm.app import App, WorkloadConfig


class CharmApp(App):
    """Go Charm application."""

    def __init__(self, *args: typing.Any, **kwargs: typing.Any) -> None:
        """Initialize the instance.

        Args:
            args: passthrough to App.
        """
        super().__init__(*args, **kwargs)

    def restart(self) -> None:
        """Restart the application."""
        self._container.add_layer(
            "charm", self._app_layer(), combine=True)
        self._prepare_service_for_restart()
        self._container.replan()


logger = logging.getLogger(__name__)


class SpecsCharm(paas_charm.go.Charm):
    """Go Charm service."""

    def __init__(self, *args: typing.Any) -> None:
        """Initialize the instance.

        Args:
            args: passthrough to CharmBase.
        """
        super().__init__(*args)

    def _create_app(self) -> App:
        """Create the application."""
        charm_state = self._create_charm_state()
        return CharmApp(
            container=self._container,
            charm_state=charm_state,
            workload_config=self._workload_config,
            database_migration=self._database_migration,
        )


if __name__ == "__main__":
    ops.main(SpecsCharm)
