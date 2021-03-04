FROM debian:bullseye-slim AS build

# make a pipe fail on the first failure
SHELL ["/bin/bash", "-o", "pipefail", "-c"]

RUN usermod --shell /bin/true --uid 1000 --home /etc/squid proxy \
  && apt-get update \
  && apt-get install --yes squid-openssl iptables curl \
  && apt-get autoremove -y && apt-get clean && rm -rf /usr/share/squid-langpack/ /var/lib/dpkg /var/cache /var/lib/dpkg /etc/apt /var/lib/apt /var/cache/apt /etc/squid \
  && mkdir -p /cache \
  && chown -R proxy:root /cache

COPY --chown=proxy docker/squid.conf /etc/squid/squid.conf
COPY --chown=proxy docker/squid.crt docker/squid.key /etc/squid/ssl/
RUN chown -R proxy:root /etc/squid \
    && chmod 600 /etc/squid/* /etc/squid/ssl/* \
    && chmod 700 /etc/squid /etc/squid/ssl

WORKDIR /etc/squid
USER proxy
EXPOSE 3129
VOLUME ["/cache"]
CMD ["sh", "-c", "/usr/lib/squid/security_file_certgen -s /cache/ssl_db -M 4MB -c && /usr/sbin/squid -f /etc/squid/squid.conf --foreground -z && exec /usr/sbin/squid -f /etc/squid/squid.conf --foreground -YCd 1"]
