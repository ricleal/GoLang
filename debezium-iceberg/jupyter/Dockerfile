# JupyterLab + GoNB (Go kernel) for the analysis layer.
#
# The analysis notebooks are GoNB notebooks (not Python): the `notebooks/`
# directory is mounted at /notebooks/work and served by JupyterLab.
#
# Based on the pre-built GoNB image (Jupyter base-notebook + Go 1.26 + gopls/
# goimports + the `gonb` kernel), pinned to a release (no `latest`), then the
# Go modules the notebooks import are pre-cached so the first cell run is fast
# and works offline.
FROM janpfeifer/gonb_jupyterlab:v0.11.5

USER root
RUN mkdir -p /notebooks/work && chown ${NB_USER}:users /notebooks/work

USER $NB_USER
# Pre-cache the modules the notebooks use. GoNB auto-runs `go get` for unknown
# imports; having them in the module cache makes that hit-free and offline.
RUN mkdir -p /tmp/nbmods && cd /tmp/nbmods && \
    go mod init nbmods >/dev/null 2>&1; \
    go get github.com/apache/iceberg-go@v0.6.0 \
           github.com/jackc/pgx/v5@v5.10.0 \
           github.com/google/uuid@latest && \
    go mod download

USER root
WORKDIR /notebooks

