FROM debian:trixie
RUN apt-get update && apt-get -y install sqlite3-tools golang python3 python-is-python3 python3-pip wget git procps
RUN useradd -d /home/cdsw cdsw
RUN pip install --break-system-packages uv
USER cdsw
COPY --chown=cdsw . /home/cdsw
WORKDIR /home/cdsw
RUN scripts/install_qdrant.sh
RUN scripts/install_py_deps.sh
RUN scripts/build_api.sh
ENV CDSW_APP_PORT=8200
ENV CDSW_API_URL=none
ENV CDSW_PROJECT_NUM=0
ENV LOCAL=true
CMD ["bash", "-c", "scripts/start_app.sh"]
