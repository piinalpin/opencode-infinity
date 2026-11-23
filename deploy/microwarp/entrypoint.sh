#!/bin/sh
# Fixture kernel WireGuard yang diadaptasi dari source MicroWARP terpin.
# shortcut: IPv4/no-auth hanya pada network fixture privat; validasi IPv6/auth saat integrasi memerlukan.
set -eu
umask 077
profile=/etc/wireguard/wg0.conf
fail() { printf '%s\n' "t04_relay_error:$1" >&2; exit 1; }
if [ "${1:-}" = write-profile ]; then
    [ ! -e "$profile" ] || fail profile_already_exists
    set -C
    cat > "$profile"
    chmod 600 "$profile"
    exit 0
fi
[ -s "$profile" ] || fail profile_required
[ "$(stat -c %a "$profile")" = 600 ] || fail profile_permissions
[ "${T04_ALLOW_NO_AUTH:-0}" = 1 ] || fail private_noauth_optin_required
[ "$(cat /proc/sys/net/ipv6/conf/all/disable_ipv6)" = 1 ] || fail ipv6_not_disabled
[ "$(cat /proc/sys/net/ipv6/conf/default/disable_ipv6)" = 1 ] || fail ipv6_default_not_disabled
address=$(sed -n 's/^Address = \([0-9.]*\/32\)$/\1/p' "$profile")
endpoint=$(sed -n 's/^Endpoint = \([0-9.]*:[0-9]*\)$/\1/p' "$profile")
[ -n "$address" ] && [ -n "$endpoint" ] || fail profile_format
peer_ip=${endpoint%:*}
peer_port=${endpoint##*:}
route=$(ip -4 route get "$peer_ip" 2>/dev/null) || fail endpoint_route
original_dev=$(printf '%s\n' "$route" | awk '{for(i=1;i<=NF;i++)if($i=="dev"){print $(i+1);exit}}')
original_gw=$(printf '%s\n' "$route" | awk '{for(i=1;i<=NF;i++)if($i=="via"){print $(i+1);exit}}')
[ -n "$original_dev" ] && [ -n "$original_gw" ] || fail endpoint_gateway
# Raw command errors/profile contents never enter logs.
ip link add wg0 type wireguard >/dev/null 2>&1 || fail kernel_wireguard
wg-quick strip "$profile" > /run/wg0.strip 2>/dev/null || fail profile_strip
wg setconf wg0 /run/wg0.strip >/dev/null 2>&1 || fail profile_load
rm -f /run/wg0.strip
ip -4 address add "$address" dev wg0 >/dev/null 2>&1 || fail interface_address
ip link set wg0 mtu 1280 >/dev/null 2>&1 || fail interface_mtu
ip -4 route replace "$peer_ip/32" via "$original_gw" dev "$original_dev" >/dev/null 2>&1 || fail endpoint_pin
# Fail closed: tidak ada blanket ESTABLISHED exception untuk egress upstream.
iptables -w 2 -P OUTPUT DROP >/dev/null 2>&1 || fail firewall_default
iptables -w 2 -A OUTPUT -o lo -j ACCEPT >/dev/null 2>&1 || fail firewall_loopback
iptables -w 2 -A OUTPUT -o wg0 -j ACCEPT >/dev/null 2>&1 || fail firewall_tunnel
iptables -w 2 -A OUTPUT -o "$original_dev" -d "$peer_ip" -p udp --dport "$peer_port" -j ACCEPT >/dev/null 2>&1 || fail firewall_endpoint
iptables -w 2 -A OUTPUT -p tcp --sport 1080 -m conntrack --ctstate ESTABLISHED --ctdir REPLY -j ACCEPT >/dev/null 2>&1 || fail firewall_socks_reply
ip -4 route replace default dev wg0 >/dev/null 2>&1 || fail tunnel_route
ip link set wg0 up >/dev/null 2>&1 || fail interface_up
printf '%s\n' t04_relay_ready
# Compose init:true memberikan PID1 yang meneruskan SIGTERM ke microsocks.
exec microsocks -i 0.0.0.0 -p 1080
