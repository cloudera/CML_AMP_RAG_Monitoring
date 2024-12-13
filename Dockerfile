FROM debian:trixie
RUN apt-get update && apt-get -y install sqlite3-tools python3 python-is-python3 python3-pip wget git procps
RUN useradd -d /home/cdsw cdsw
RUN pip install --break-system-packages uv
USER cdsw
COPY --chown=cdsw . /home/cdsw
WORKDIR /home/cdsw
ENV CDSW_APP_PORT=8100
ENV CDSW_API_URL=none
ENV CDSW_PROJECT_NUM=0
ENV LOCAL=true
CMD ["bash", "-c", "scripts/install_qdrant.sh && scripts/install_golang.sh && scripts/install_py_deps.sh && scripts/build_api.sh && scripts/start_app.sh"]
