#!/bin/bash

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

CANDIDATE_PORTS=(17463 23891 34572 41239 52847 44821 38456 29173 56392 47215)
LOG_FILE="/tmp/hr-deploy.log"

log()    { echo -e "${BLUE}[INFO]${NC}  $*" | tee -a "$LOG_FILE"; }
ok()     { echo -e "${GREEN}[OK]${NC}    $*" | tee -a "$LOG_FILE"; }
warn()   { echo -e "${YELLOW}[WARN]${NC}  $*" | tee -a "$LOG_FILE"; }
err()    { echo -e "${RED}[ERROR]${NC} $*" | tee -a "$LOG_FILE"; }
banner() { echo -e "${CYAN}$*${NC}"; }

echo "" > "$LOG_FILE"

banner "
╔══════════════════════════════════════╗
║   HR Boshqaruv Tizimi - Deploy       ║
║   Hikvision Davomat Tizimi           ║
╚══════════════════════════════════════╝
"

# ─────────────────────────────────────
# 1. Docker o'rnatish
# ─────────────────────────────────────
install_docker() {
    warn "Docker o'rnatilmoqda..."
    if command -v apt-get >/dev/null 2>&1; then
        apt-get update -qq 2>>"$LOG_FILE"
        apt-get install -y -qq ca-certificates curl gnupg 2>>"$LOG_FILE"
        install -m 0755 -d /etc/apt/keyrings
        curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
            | gpg --dearmor -o /etc/apt/keyrings/docker.gpg 2>/dev/null
        chmod a+r /etc/apt/keyrings/docker.gpg
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
https://download.docker.com/linux/ubuntu $(lsb_release -cs 2>/dev/null || echo jammy) stable" \
            > /etc/apt/sources.list.d/docker.list
        apt-get update -qq 2>>"$LOG_FILE"
        apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin 2>>"$LOG_FILE"
    elif command -v yum >/dev/null 2>&1; then
        yum install -y -q yum-utils 2>>"$LOG_FILE"
        yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
        yum install -y -q docker-ce docker-ce-cli containerd.io docker-compose-plugin 2>>"$LOG_FILE"
    else
        curl -fsSL https://get.docker.com | sh >>"$LOG_FILE" 2>&1
    fi
    systemctl enable docker 2>/dev/null || true
    systemctl start docker 2>/dev/null || true
    sleep 3
}

if ! command -v docker >/dev/null 2>&1; then
    install_docker
fi

if ! docker info >/dev/null 2>&1; then
    warn "Docker daemon ishlamayapti, qayta ishga tushirilmoqda..."
    systemctl start docker 2>/dev/null || service docker start 2>/dev/null || true
    sleep 5
fi

if ! docker info >/dev/null 2>&1; then
    err "Docker ishga tushmadi. Loglar: $LOG_FILE"
    exit 1
fi
ok "Docker: $(docker --version)"

# ─────────────────────────────────────
# 2. Docker Compose
# ─────────────────────────────────────
if docker compose version >/dev/null 2>&1; then
    COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
    COMPOSE="docker-compose"
else
    warn "Docker Compose o'rnatilmoqda..."
    COMPOSE_URL="https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)"
    curl -SL "$COMPOSE_URL" -o /usr/local/bin/docker-compose 2>>"$LOG_FILE"
    chmod +x /usr/local/bin/docker-compose
    COMPOSE="docker-compose"
fi
ok "Compose: $($COMPOSE version --short 2>/dev/null || echo 'ok')"

# ─────────────────────────────────────
# 3. Bosh port topish
# ─────────────────────────────────────
is_port_free() {
    local port=$1
    # Try multiple methods
    if command -v ss >/dev/null 2>&1; then
        ss -tln | grep -q ":${port} " && return 1
    elif command -v netstat >/dev/null 2>&1; then
        netstat -tln | grep -q ":${port} " && return 1
    else
        (echo >/dev/tcp/127.0.0.1/"$port") 2>/dev/null && return 1
    fi
    # Also check docker
    docker ps --format "{{.Ports}}" 2>/dev/null | grep -q ":${port}->" && return 1
    return 0
}

# ─────────────────────────────────────
# 4. .env fayl
# ─────────────────────────────────────
if [ -f .env ]; then
    warn ".env mavjud, ishlatilmoqda..."
    set -a; source .env; set +a

    if [ -z "${APP_PORT:-}" ]; then
        for port in "${CANDIDATE_PORTS[@]}"; do
            if is_port_free "$port"; then
                APP_PORT="$port"
                echo "APP_PORT=${APP_PORT}" >> .env
                warn "APP_PORT qo'shildi: ${APP_PORT}"
                break
            fi
        done
    fi
    if [ -z "${SESSION_SECRET:-}" ]; then
        SESSION_SECRET=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | head -c 48)
        echo "SESSION_SECRET=${SESSION_SECRET}" >> .env
    fi
else
    log ".env yaratilmoqda..."
    for port in "${CANDIDATE_PORTS[@]}"; do
        if is_port_free "$port"; then
            APP_PORT="$port"
            break
        fi
    done

    if [ -z "${APP_PORT:-}" ]; then
        err "Barcha 10 port band. /tmp/hr-deploy.log ga qarang."
        exit 1
    fi

    SESSION_SECRET=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | head -c 48)
    POSTGRES_PASSWORD=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | head -c 24)

    cat > .env <<EOF
APP_PORT=${APP_PORT}
SESSION_SECRET=${SESSION_SECRET}
POSTGRES_DB=hr_system
POSTGRES_USER=hr_user
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
EOF
    ok ".env yaratildi"
fi

set -a; source .env; set +a
ok "Port: ${APP_PORT}"

# ─────────────────────────────────────
# 5. Eski konteynerlarni to'xtatish
# ─────────────────────────────────────
if $COMPOSE ps 2>/dev/null | grep -qE "Up|running|Exit"; then
    warn "Eski konteynerlar to'xtatilmoqda..."
    $COMPOSE down --remove-orphans --timeout 30 2>>"$LOG_FILE" || true
    sleep 3
    ok "Eski konteynerlar to'xtatildi"
fi

# ─────────────────────────────────────
# 6. Build
# ─────────────────────────────────────
log "Image build qilinmoqda (10-20 daqiqa ketishi mumkin)..."
log "Log fayli: $LOG_FILE"

BUILD_OK=0
for attempt in 1 2 3; do
    log "Build urinish $attempt/3..."
    if $COMPOSE build --no-cache >> "$LOG_FILE" 2>&1; then
        BUILD_OK=1
        ok "Build muvaffaqiyatli (urinish $attempt)"
        break
    else
        err "Build xatosi (urinish $attempt/3). Sabab:"
        tail -20 "$LOG_FILE" | grep -E "error|Error|ERROR|failed|FAILED" | head -10 || true
        if [ "$attempt" -lt 3 ]; then
            warn "30 soniyadan keyin qayta uriniladi..."
            sleep 30
        fi
    fi
done

if [ "$BUILD_OK" -eq 0 ]; then
    err "Build 3 urinishdan keyin ham muvaffaqiyatsiz tugadi!"
    err "To'liq loglar: $LOG_FILE"
    echo ""
    err "=== OXIRGI 50 QATOR LOG ==="
    tail -50 "$LOG_FILE"
    exit 1
fi

# ─────────────────────────────────────
# 7. Ishga tushirish
# ─────────────────────────────────────
log "Konteynerlar ishga tushirilmoqda..."
if ! $COMPOSE up -d >> "$LOG_FILE" 2>&1; then
    err "Konteynerlar ishga tushmadi"
    $COMPOSE logs --tail=50
    exit 1
fi

# ─────────────────────────────────────
# 8. Sog'liqni tekshirish
# ─────────────────────────────────────
log "Tizim tayyorlanmoqda (max 120 soniya)..."

READY=0
for i in $(seq 1 24); do
    sleep 5
    UP_COUNT=$($COMPOSE ps 2>/dev/null | grep -cE "Up|running" || echo 0)
    if [ "$UP_COUNT" -ge 2 ]; then
        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
            "http://127.0.0.1:${APP_PORT}/" \
            --max-time 5 --connect-timeout 3 2>/dev/null || echo "000")
        if [[ "$HTTP_CODE" =~ ^(200|301|302|304)$ ]]; then
            READY=1
            ok "Tizim ishlayapti! HTTP ${HTTP_CODE}"
            break
        fi
        log "Kutilmoqda... konteynerlar: $UP_COUNT, HTTP: $HTTP_CODE (${i}/24)"
    else
        log "Konteynerlar ishga tushmoqda... ($UP_COUNT tayyor, ${i}/24)"
    fi
done

# ─────────────────────────────────────
# 9. Konteynerlar to'xta qolsa — restart
# ─────────────────────────────────────
if [ "$READY" -eq 0 ]; then
    warn "Konteynerlar javob bermayapti, restart qilinmoqda..."
    $COMPOSE restart 2>>"$LOG_FILE" || true
    sleep 20

    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
        "http://127.0.0.1:${APP_PORT}/" \
        --max-time 10 --connect-timeout 5 2>/dev/null || echo "000")

    if [[ "$HTTP_CODE" =~ ^(200|301|302|304)$ ]]; then
        READY=1
        ok "Restart dan keyin ishlayapti! HTTP ${HTTP_CODE}"
    else
        err "Tizim restart dan keyin ham javob bermadi (HTTP: $HTTP_CODE)"
        err ""
        err "Konteyner holati:"
        $COMPOSE ps
        err ""
        err "App loglari:"
        $COMPOSE logs app --tail=30 2>/dev/null || $COMPOSE logs --tail=30
        err ""
        err "To'liq loglar: $LOG_FILE"
        exit 1
    fi
fi

# ─────────────────────────────────────
# 10. Natija
# ─────────────────────────────────────
SERVER_IP=$(curl -s --max-time 5 ifconfig.me 2>/dev/null \
    || curl -s --max-time 5 api.ipify.org 2>/dev/null \
    || hostname -I 2>/dev/null | awk '{print $1}' \
    || echo "SERVER_IP")

banner "
╔═══════════════════════════════════════════════════╗
║      HR TIZIM MUVAFFAQIYATLI ISHGA TUSHDI!       ║
╠═══════════════════════════════════════════════════╣
║  URL   : http://${SERVER_IP}:${APP_PORT}
║  Login : sudo  / sudo123                          ║
║  Login : admin / admin123                         ║
╠═══════════════════════════════════════════════════╣
║  Subdomain uchun reverse proxy:                   ║
║  Target: http://127.0.0.1:${APP_PORT}             ║
╚═══════════════════════════════════════════════════╝

Komandalar:
  Loglar  : $COMPOSE logs -f
  To'xtat : $COMPOSE down
  Restart : $COMPOSE restart
  Holat   : $COMPOSE ps
"
