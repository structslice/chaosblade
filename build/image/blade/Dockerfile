FROM thub.autohome.com.cn/ops-cache/cache-centos:7
LABEL maintainer="FuZZ"

# The image is chaosblade image
ARG BLADE_VERSION=0.0.1

ENV CHAOSBLADE_HOME /opt/chaosblade
WORKDIR ${CHAOSBLADE_HOME}

COPY chaosblade-${BLADE_VERSION} ${CHAOSBLADE_HOME}

ENV PATH ${CHAOSBLADE_HOME}:${PATH}
CMD ["sh", "-c", "tail -f /dev/null"]
