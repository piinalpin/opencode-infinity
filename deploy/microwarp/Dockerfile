# Source referensi MicroWARP MIT dipin; runtime fixture memakai startup teredaksi.
FROM docker.io/library/alpine:3.24.4@sha256:0f49f331d9212aff11b932863be4eb01cdf962969aa86dab7d1ee21ac6f5f52b AS microsocks-builder
RUN apk add --no-cache build-base=0.5-r4 git=2.54.1-r0
RUN git init /src && git -C /src remote add origin https://github.com/rofl0r/microsocks.git \
    && git -C /src fetch --depth 1 origin 98421a21c4adc4c77c0cf3a5d650cc28ad3e0107 \
    && git -C /src checkout --detach FETCH_HEAD && make -C /src
FROM docker.io/library/alpine:3.24.4@sha256:0f49f331d9212aff11b932863be4eb01cdf962969aa86dab7d1ee21ac6f5f52b
RUN apk add --no-cache wireguard-tools=1.0.20260223-r0 iptables=1.8.13-r0 iproute2=7.0.0-r0
COPY --from=microsocks-builder /src/microsocks /usr/local/bin/microsocks
COPY entrypoint.sh /usr/local/bin/t04-entrypoint
COPY LICENSE.warp-relay-pool LICENSE.microsocks /usr/share/licenses/t04/
RUN chmod 755 /usr/local/bin/t04-entrypoint && mkdir -p /etc/wireguard && chmod 700 /etc/wireguard
LABEL opencode-infinity.task="EP-01-T04" \
      opencode-infinity.reference.commit="1a58c904d6a99463f2c48cb02234f44eca7646e6" \
      opencode-infinity.microsocks.commit="98421a21c4adc4c77c0cf3a5d650cc28ad3e0107"
STOPSIGNAL SIGTERM
ENTRYPOINT ["/usr/local/bin/t04-entrypoint"]
