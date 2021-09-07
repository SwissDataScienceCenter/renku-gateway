Renku Gateway Helm Chart
========================

Provide a basic chart for deploying the Renku Gateway application.

Configuration
-------------

- `gitlab.url` define the URL of a running GitLab instance
  (default: `http://gitlab.renku.build`)
- `keycloakUrl` define the URL of a running JupyterHub instance
  (default: `http://keycloak.renku.build:8080`)

Usage
-----

In the `helm-chart` directory:

.. code-block:: console

    helm upgrade --install renku-gateway --values minikube-values.yaml renku-gateway


To rebuild the images and update the chart you can run

.. code-block:: console

    pip install chartpress
    chartpress
