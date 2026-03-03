#!/bin/bash

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

CANDIDATE_PORTS=(17463 23891 34572 41239 52847 44821 38456 29173 56392 47215)

log()    { echo -e "${BLUE}[INFO]${NC}  $*"; }
ok()     { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()   { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error()  { echo -e "${RED}[ERROR]${NC} $*"; }
banner() { echo -e "${CYAN}$*${NC}"; }

banner "
╔══════════════════════════════════════╗
║   HR Boshqaruv Tizimi - Deploy       ║
║   Hikvision Davomat Tizimi           ║
╚══════════════════════════════════════╝
"

#───────────────────────────────────────
# 1. Docker tekshirish / o'rnatish
#───────────────────────────────────────
install_docker() {
    warn "Docker topilmadi. O'rnatilmoqda..."
    if command -v apt-get &>/dev/null; then
        apt-get update -qq
        apt-get install -y -qq ca-certificates curl gnupg lsb-release
        install -m 0755 -d /etc/apt/keyrings
        curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
            | gpg --dearmor -o /etc/apt/keyrings/docker.gpg 2>/dev/null
        chmod a+r /etc/apt/keyrings/docker.gpg
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" \
            > /etc/apt/sources.list.d/docker.list
        apt-get update -qq
        apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    elif command -v yum &>/dev/null; then
        yum install -y -q yum-utils
        yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
        yum install -y -q docker-ce docker-ce-cli containerd.io docker-compose-plugin
    else
        curl -fsSL https://get.docker.com | sh
    fi
    systemctl enable docker 2>/dev/null || true
    systemctl start docker 2>/dev/null || true
    ok "Docker o'rnatildi"
}

if ! command -v docker &>/dev/null; then
    install_docker
else
    ok "Docker mavjud: $(docker --version)"
fi

#───────────────────────────────────────
# 2. Docker Compose komandasi aniqlash
#───────────────────────────────────────
if docker compose version &>/dev/null 2>&1; then
    COMPOSE="docker compose"
    ok "Docker Compose: $(docker compose version --short 2>/dev/null || echo 'ok')"
elif command -v docker-compose &>/dev/null; then
    COMPOSE="docker-compose"
    ok "docker-compose mavjud"
else
    warn "Docker Compose o'rnatilmoqda..."
    curl -SL "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" \
        -o /usr/local/bin/docker-compose
    chmod +x /usr/local/bin/docker-compose
    COMPOSE="docker-compose"
    ok "docker-compose o'rnatildi"
fi

#───────────────────────────────────────
# 3. Bosh port topish
#───────────────────────────────────────
find_free_port() {
    for port in "${CANDIDATE_PORTS[@]}"; do
        if ! (ss -tln 2>/dev/null || netstat -tln 2>/dev/null) | grep -q ":${port}[[:space:]]"; then
            if ! docker ps --format "{{.Ports}}" 2>/dev/null | grep -q ":${port}->"; then
                echo "$port"
                return 0
            fi
        fi
    done
    return 1
}

#───────────────────────────────────────
# 4. .env fayl yaratish / yangilash
#───────────────────────────────────────
if [ -f .env ]; then
    warn ".env fayl mavjud, o'qilmoqda..."
    source .env 2>/dev/null || true

    if [ -z "${APP_PORT:-}" ]; then
        APP_PORT=$(find_free_port) || { error "Barcha portlar band!"; exit 1; }
        echo "APP_PORT=${APP_PORT}" >> .env
        warn "APP_PORT .env ga qo'shildi: ${APP_PORT}"
    fi

    if [ -z "${SESSION_SECRET:-}" ]; then
        SESSION_SECRET=$(head -c 48 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 48)
        echo "SESSION_SECRET=${SESSION_SECRET}" >> .env
        warn "SESSION_SECRET .env ga qo'shildi"
    fi
else
    log ".env fayl yaratilmoqda..."

    APP_PORT=$(find_free_port) || { error "Barcha portlar band!"; exit 1; }
    SESSION_SECRET=$(head -c 48 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 48)
    POSTGRES_PASS=$(head -c 24 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 24)

    cat > .env <<EOF
APP_PORT=${APP_PORT}
SESSION_SECRET=${SESSION_SECRET}
POSTGRES_DB=hr_system
POSTGRES_USER=hr_user
POSTGRES_PASSWORD=${POSTGRES_PASS}
EOF
    ok ".env fayl yaratildi (port: ${APP_PORT})"
fi

source .env
APP_PORT="${APP_PORT}"

ok "Tanlangan port: ${APP_PORT}"

#───────────────────────────────────────
# 5. Eski konteynerlarni to'xtatish
#───────────────────────────────────────
RUNNING=$($COMPOSE ps --services --filter "status=running" 2>/dev/null || echo "")
if [ -n "$RUNNING" ]; then
    warn "Eski konteynerlar to'xtatilmoqda..."
    $COMPOSE down --remove-orphans 2>/dev/null || true
    ok "Eski konteynerlar to'xtatildi"
fi

#───────────────────────────────────────
# 6. Build qilish
#───────────────────────────────────────
log "Docker image build qilinmoqda (5-15 daqiqa ketishi mumkin)..."
log "Sabr qiling..."

BUILD_ATTEMPTS=0
MAX_BUILD_ATTEMPTS=3

while [ $BUILD_ATTEMPTS -lt $MAX_BUILD_ATTEMPTS ]; do
    BUILD_ATTEMPTS=$((BUILD_ATTEMPTS + 1))

    if $COMPOSE build --no-cache 2>&1 | tee /tmp/hr_build.log | grep -E "^(Step|#|Successfully|ERROR|error)" ; then
        ok "Build muvaffaqiyatli tugadi"
        break
    else
        BUILD_EXIT=${PIPESTATUS[0]}
        if [ $BUILD_EXIT -ne 0 ] && [ $BUILD_ATTEMPTS -lt $MAX_BUILD_ATTEMPTS ]; then
            warn "Build xatoligi, qayta urinilmoqda ($BUILD_ATTEMPTS/$MAX_BUILD_ATTEMPTS)..."
            sleep 5
        elif [ $BUILD_EXIT -ne 0 ]; then
            error "Build $MAX_BUILD_ATTEMPTS marta urinishdan keyin ham muvaffaqiyatsiz:"
            tail -30 /tmp/hr_build.log
            exit 1
        else
            ok "Build muvaffaqiyatli tugadi"
            break
        fi
    fi
done

#───────────────────────────────────────
# 7. Ishga tushirish
#───────────────────────────────────────
log "Konteynerlar ishga tushirilmoqda..."

$COMPOSE up -d 2>&1

#───────────────────────────────────────
# 8. Tayyorligini kutish va tekshirish
#───────────────────────────────────────
log "Tizim tayyorlanmoqda, iltimos kuting..."

MAX_WAIT=90
ELAPSED=0
INTERVAL=5

while [ $ELAPSED -lt $MAX_WAIT ]; do
    sleep $INTERVAL
    ELAPSED=$((ELAPSED + INTERVAL))

    APP_STATUS=$($COMPOSE ps --format json 2>/dev/null | grep -c '"State":"running"' || \
                 $COMPOSE ps 2>/dev/null | grep -c "Up" || echo "0")

    if [ "$APP_STATUS" -ge 2 ]; then
        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
            "http://127.0.0.1:${APP_PORT}/" \
            --max-time 5 --connect-timeout 5 2>/dev/null || echo "000")

        if [[ "$HTTP_CODE" == "200" || "$HTTP_CODE" == "302" || "$HTTP_CODE" == "301" ]]; then
            ok "Tizim javob bermoqda (HTTP ${HTTP_CODE})"
            break
        fi
        log "HTTP javob kutilmoqda... (${ELAPSED}/${MAX_WAIT}s)"
    else
        log "Konteynerlar ishga tushmoqda... (${ELAPSED}/${MAX_WAIT}s)"
    fi
done

#───────────────────────────────────────
# 9. Yakuniy holat tekshirish
#───────────────────────────────────────
log "Yakuniy tekshiruv..."
$COMPOSE ps

CONTAINERS_UP=$($COMPOSE ps 2>/dev/null | grep -c "Up\|running" || echo "0")

if [ "$CONTAINERS_UP" -lt 2 ]; then
    error "Ba'zi konteynerlar ishlamayapti. Loglar:"
    $COMPOSE logs --tail=50

    warn "Qayta ishga tushirilmoqda..."
    $COMPOSE restart 2>/dev/null || true
    sleep 15

    CONTAINERS_UP=$($COMPOSE ps 2>/dev/null | grep -c "Up\|running" || echo "0")
    if [ "$CONTAINERS_UP" -lt 2 ]; then
        error "Tizim ishga tushmadi. Batafsil loglar:"
        $COMPOSE logs
        exit 1
    fi
fi

#───────────────────────────────────────
# 10. Natija
#───────────────────────────────────────
SERVER_IP=$(curl -s --max-time 5 ifconfig.me 2>/dev/null \
    || curl -s --max-time 5 api.ipify.org 2>/dev/null \
    || hostname -I 2>/dev/null | awk '{print $1}' \
    || echo "SERVER_IP")

banner "
╔══════════════════════════════════════════════════╗
║        HR TIZIM MUVAFFAQIYATLI ISHGA TUSHDI!    ║
╠══════════════════════════════════════════════════╣
║  URL     : http://${SERVER_IP}:${APP_PORT}
║  Login   : sudo    / sudo123                     ║
║  Login   : admin   / admin123                    ║
╠══════════════════════════════════════════════════╣
║  Subdomain ulash uchun:                          ║
║  DNS A-record → ${SERVER_IP}             ║
║  Nginx/Apache proxy → port ${APP_PORT}           ║
╚══════════════════════════════════════════════════╝

Foydali komandalar:
  Loglar ko'rish : $COMPOSE logs -f
  To'xtatish     : $COMPOSE down
  Qayta boshlash : $COMPOSE restart
  Holat          : $COMPOSE ps
"
