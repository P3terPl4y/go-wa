package handlers

import (
	"App/src/controllers/get"
	"fmt"

	"github.com/gofiber/fiber/v3"
)

func Dashboard(c fiber.Ctx) error {
app.Get("/dashboard", authRequired, func(c fiber.Ctx) error {
	userID := c.Locals("user_id").(int)
	role := c.Locals("role").(string)
	user, err := getUserByID(userID)
	if err != nil || user == nil {
		return c.Status(500).JSON(fiber.Map{"error": "Usuario no encontrado"})
	}
	bots, err := getBotsByUser(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Error al obtener bots"})
	}
	var botID int
	var botInfo string
	var currentPrompt string
	var paymentStatus string
	if len(bots) == 0 {
		botInfo = "No tienes ningún bot. Crea uno desde aquí."
	} else {
		bot := bots[0]
		botID = bot.ID
		paymentStatus = bot.PaymentStatus
		botInfo = fmt.Sprintf("Bot ID: %d | Bloqueado: %v | Pago: %s", bot.ID, bot.Blocked, bot.PaymentStatus)
		prompt, _ := GetPrompt(bot.ID)
		currentPrompt = prompt
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Dashboard · Wago</title>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.0.0-beta3/css/all.min.css" />
    <link rel="stylesheet" href="/assets/css/styles.css?v=3" />
    <style>
        .dashboard-container { max-width: 900px; margin: 40px auto; padding: 0 24px; }
        .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 32px; flex-wrap: wrap; gap: 16px; }
        .brand { display: flex; align-items: center; gap: 12px; }
        .brand img { height: 44px; width: 44px; border-radius: 50%; object-fit: cover; }
        .brand span { font-size: 1.6rem; font-weight: 700; color: var(--color-text-main); }
        .brand span small { font-weight: 400; color: var(--color-text-muted); font-size: 1rem; }
        .user-badge { display: flex; align-items: center; gap: 12px; background: var(--color-bg-surface-elevated); padding: 6px 16px 6px 6px; border-radius: 40px; border: 1px solid var(--color-border); }
        .user-avatar { width: 36px; height: 36px; border-radius: 50%; background: var(--color-primary); display: flex; align-items: center; justify-content: center; font-weight: 700; font-size: 16px; color: #000; flex-shrink: 0; }
        .user-name { font-weight: 600; font-size: 14px; color: var(--color-text-main); }
        .user-role { font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; color: var(--color-primary); background: rgba(16, 185, 129, 0.1); padding: 4px 10px; border-radius: 20px; }
        .card-header { display: flex; align-items: center; gap: 12px; margin-bottom: 24px; }
        .card-header .icon { font-size: 20px; color: var(--color-primary); width: 32px; height: 32px; background: rgba(16, 185, 129, 0.1); border-radius: 8px; display: flex; align-items: center; justify-content: center; }
        .card-header h3 { font-size: 1.25rem; font-weight: 600; margin: 0; }
        .user-info-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 16px; }
        .user-info-item { background: var(--color-bg-base); border-radius: 12px; padding: 16px; border: 1px solid var(--color-border); }
        .user-info-item .label { font-size: 11px; text-transform: uppercase; letter-spacing: 0.5px; color: var(--color-text-muted); font-weight: 600; margin-bottom: 4px; }
        .user-info-item .value { font-size: 15px; font-weight: 500; color: var(--color-text-main); word-break: break-all; }
        .bot-status-row { display: flex; align-items: center; gap: 16px; margin-bottom: 24px; }
        .bot-status-indicator { display: flex; align-items: center; gap: 8px; font-size: 14px; font-weight: 600; padding: 6px 16px; border-radius: 30px; background: var(--color-bg-base); border: 1px solid var(--color-border); }
        .bot-status-dot { width: 10px; height: 10px; border-radius: 50%; display: inline-block; }
        .bot-status-dot.inactive { background: var(--color-text-muted); }
        .bot-status-dot.active { background: var(--color-primary); box-shadow: 0 0 12px var(--color-primary); }
        .bot-status-dot.error { background: var(--color-error); box-shadow: 0 0 12px var(--color-error); }
        .bot-status-dot.qr { background: var(--color-warning); box-shadow: 0 0 12px var(--color-warning); animation: pulse-dot 1.5s infinite; }
        @keyframes pulse-dot { 0%, 100% { opacity: 1; transform: scale(1); } 50% { opacity: 0.5; transform: scale(0.8); } }
        .qr-wrapper { margin-top: 24px; padding: 24px; background: var(--color-bg-base); border-radius: 16px; border: 1px solid var(--color-border); display: flex; flex-direction: column; align-items: center; gap: 16px; }
        .qr-wrapper img { border-radius: 12px; background: #fff; padding: 12px; max-width: 240px; width: 100%; height: auto; }
        .my-4 { margin: 24px 0; }
        @media(max-width: 640px) { .header { flex-direction: column; align-items: flex-start; } .user-badge { width: 100%; justify-content: center; } .btn-group { flex-direction: column; } .btn-group .btn { width: 100%; } }
    </style>
</head>
<body>
<div class="app-layout">
    <!-- Sidebar -->
    <aside class="sidebar">
        <div class="sidebar-header">
            <img src="https://copilot.microsoft.com/shares/XsBodgkpbrmLF5JjNvXrA" alt="Wago" style="width:36px;height:36px;border-radius:50%;" />
            <span style="font-size:1.25rem;font-weight:700;color:var(--color-text-main);">Wago <small style="font-weight:400;color:var(--color-text-muted);font-size:0.8rem;">Panel</small></span>
        </div>
        <nav class="sidebar-nav">
            <a href="#perfil" class="sidebar-item active"><i class="fas fa-user"></i> Perfil</a>
            <a href="#bot" class="sidebar-item"><i class="fas fa-robot"></i> Tu Bot</a>
            <a href="#prompt" class="sidebar-item"><i class="fas fa-cog"></i> Prompt</a>
            <a href="#seguridad" class="sidebar-item"><i class="fas fa-shield-alt"></i> Seguridad</a>
        </nav>
        <div class="sidebar-footer" style="display:flex; justify-content:space-between; align-items:center;">
            <button class="theme-toggle" id="themeToggleDash" aria-label="Cambiar tema">
                <i class="fas fa-moon"></i>
            </button>
            <button class="btn btn-danger btn-sm" id="logoutBtn" style="padding:6px 12px;"><i class="fas fa-sign-out-alt"></i> Salir</button>
        </div>
    </aside>

    <!-- Main Content -->
    <main class="main-content">
        <div class="topbar">
            <div>
                <div class="breadcrumb">Inicio <i class="fas fa-chevron-right" style="font-size:0.6rem;"></i> Panel</div>
                <h1 class="page-title">Mi Cuenta</h1>
            </div>
            <div class="header-actions">
                <div class="user-badge" style="display:flex; align-items:center; gap:12px; background:var(--color-bg-surface-elevated); padding:6px 16px 6px 6px; border-radius:40px; border:1px solid var(--color-border);">
                    <div class="user-avatar" id="avatarLetter" style="width:36px; height:36px; border-radius:50%; background:var(--color-primary); display:flex; align-items:center; justify-content:center; font-weight:700; font-size:16px; color:#000;">U</div>
                    <span class="user-name" id="displayName" style="font-weight:600; font-size:14px; color:var(--color-text-main);">Usuario</span>
                    <span class="user-role" id="displayRole" style="font-size:11px; font-weight:700; text-transform:uppercase; letter-spacing:0.5px; color:var(--color-primary); background:rgba(16,185,129,0.1); padding:4px 10px; border-radius:20px;">Rol</span>
                </div>
            </div>
        </div>

        <!-- Perfil -->
        <div id="perfil" class="premium-card glass-card mb-4 reveal">
            <div class="premium-card-body">
                <div class="card-header"><div class="icon"><i class="fas fa-user"></i></div><h3>Perfil</h3></div>
                <div class="user-info-grid">
                    <div class="user-info-item"><div class="label">Usuario</div><div class="value" id="infoUser">—</div></div>
                    <div class="user-info-item"><div class="label">Email</div><div class="value" id="infoEmail">—</div></div>
                    <div class="user-info-item"><div class="label">Teléfono</div><div class="value" id="infoPhone">—</div></div>
                    <div class="user-info-item"><div class="label">Rol</div><div class="value" id="infoRole">—</div></div>
                </div>
            </div>
        </div>

        <!-- Bot -->
        <div id="bot" class="premium-card glass-card mb-4 reveal delay-1">
            <div class="premium-card-body">
                <div class="card-header justify-between w-full">
                    <div class="flex items-center gap-3"><div class="icon"><i class="fas fa-robot"></i></div><h3>Tu Bot</h3></div>
                    <span class="badge badge-inactive" id="botIdBadge">ID: —</span>
                </div>
                <div class="bot-status-row">
                    <span class="text-muted text-sm font-semibold uppercase">Estado Actual</span>
                    <div class="bot-status-indicator">
                        <span class="bot-status-dot inactive" id="botStatusDot"></span>
                        <span id="botStatusText">Inactivo</span>
                    </div>
                </div>
                <div id="paymentStatusMsg" style="margin-bottom:20px;font-size:14px;color:var(--color-warning);"></div>
                <div class="flex gap-3 btn-group">
                    <button class="btn btn-primary" id="startBotBtn"><i class="fas fa-play"></i> Iniciar Bot</button>
                    <button class="btn btn-secondary" id="refreshStatusBtn"><i class="fas fa-sync-alt"></i> Actualizar</button>
                </div>
                <div id="qrContainer" class="qr-wrapper" style="display:none;">
                    <img id="qrImage" src="" alt="Código QR" />
                    <span class="text-muted text-sm font-medium"><i class="fas fa-qrcode"></i> Escanea con WhatsApp</span>
                </div>
                <div id="status" class="status-msg hidden"></div>
            </div>
        </div>

        <!-- Prompt -->
        <div id="prompt" class="premium-card glass-card mb-4 reveal delay-2">
            <div class="premium-card-body">
                <div class="card-header"><div class="icon"><i class="fas fa-cog"></i></div><h3>Configurar Prompt</h3></div>
                <div class="form-group">
                    <label for="promptInput">Prompt (define el comportamiento del bot)</label>
                    <textarea class="form-control" id="promptInput" placeholder="Ej: Eres un asistente amable que responde en español…"></textarea>
                </div>
                <button class="btn btn-primary" id="updatePromptBtn"><i class="fas fa-save"></i> Guardar Prompt</button>
                <div id="promptStatus" class="status-msg hidden"></div>
            </div>
        </div>

        <!-- Seguridad -->
        <div id="seguridad" class="premium-card glass-card mb-4 reveal delay-3">
            <div class="premium-card-body">
                <div class="card-header"><div class="icon"><i class="fas fa-key"></i></div><h3>Seguridad</h3></div>

                <!-- Actualizar Teléfono -->
                <div class="form-group">
                    <label for="newPhone">Nuevo número de teléfono (formato internacional)</label>
                    <input type="text" class="form-control" id="newPhone" placeholder="+521234567890" />
                </div>
                <button class="btn btn-primary" id="updatePhoneBtn"><i class="fas fa-phone"></i> Actualizar Teléfono</button>
                <div id="phoneStatus" class="status-msg hidden"></div>

                <hr class="my-4" />

                <!-- Actualizar Contraseña -->
                <div class="form-group">
                    <label for="newPass">Nueva contraseña</label>
                    <input type="password" class="form-control" id="newPass" placeholder="Ingresa tu nueva contraseña…" />
                </div>
                <button class="btn btn-primary" id="changePassBtn"><i class="fas fa-lock"></i> Actualizar</button>
                <div id="passStatus" class="status-msg hidden"></div>
            </div>
        </div>

        <div class="flex justify-between items-center mt-4 mb-4">
            <span class="text-muted text-xs">Wago Panel &copy; 2026</span>
        </div>
    </main>
</div>

<script>
(function(){
    const $=id=>document.getElementById(id);
    let botID = window.botID || 0;
    const userDisplay = window.userDisplay || 'Usuario';
    const userEmail = window.userEmail || 'usuario@email.com';
    const userPhone = window.userPhone || '—';
    const userRole = window.userRole || 'usuario';
    let paymentStatus = window.paymentStatus || 'free';

    $('displayName').textContent = userDisplay;
    $('displayRole').textContent = userRole;
    $('avatarLetter').textContent = userDisplay.charAt(0).toUpperCase();
    $('infoUser').textContent = userDisplay;
    $('infoEmail').textContent = userEmail;
    $('infoPhone').textContent = userPhone;
    $('infoRole').textContent = userRole;
    if (botID) $('botIdBadge').textContent = 'ID: ' + botID;

    function showStatus(el, msg, type){
        if(!el) return;
        el.className = 'status-msg '+(type||'info');
        el.textContent = msg;
        el.classList.remove('hidden');
    }
    function hideStatus(el){ if(el) el.classList.add('hidden'); }
    function setBotStatus(state, text){
        const dot=$('botStatusDot'), label=$('botStatusText');
        if(!dot||!label) return;
        dot.className='bot-status-dot';
        if(state==='active'){dot.classList.add('active');label.textContent=text||'Activo';}
        else if(state==='qr'){dot.classList.add('qr');label.textContent=text||'Escanea QR';}
        else if(state==='error'){dot.classList.add('error');label.textContent=text||'Error';}
        else {dot.classList.add('inactive');label.textContent=text||'Inactivo';}
    }
    function showQR(base64){
        const container=$('qrContainer'), img=$('qrImage');
        if(!container||!img) return;
        if(base64){img.src='data:image/png;base64,'+base64;container.style.display='flex';setBotStatus('qr','Escanea el QR');}
        else container.style.display='none';
    }
    function hideQR(){ const c=$('qrContainer'); if(c) c.style.display='none'; }

    function updatePaymentStatusMsg(status) {
        const el = $('paymentStatusMsg');
        if (!el) return;
        if (status === 'pending') {
            el.innerHTML = '<i class="fas fa-clock"></i> Pago pendiente. Espera la confirmación del administrador.';
            el.style.color = '#e67e22';
        } else {
            el.innerHTML = '';
        }
        const startBtn = $('startBotBtn');
        if (startBtn) {
            if (status === 'pending') {
                startBtn.disabled = true;
                startBtn.innerHTML = '<i class="fas fa-hourglass-half"></i> Esperando pago...';
            } else {
                startBtn.disabled = false;
                startBtn.innerHTML = '<i class="fas fa-play"></i> Iniciar Bot';
            }
        }
    }

    if (window.paymentStatus) {
        updatePaymentStatusMsg(window.paymentStatus);
    }

    // Iniciar Bot
    $('startBotBtn').addEventListener('click', function(){
        const status=$('status'); hideStatus(status); hideQR();
        showStatus(status, '⏳ Procesando...', 'info');
        fetch('/start-bot', {method:'POST', headers:{'Content-Type':'application/json'}, body:'{}'})
        .then(r=>r.json()).then(d=>{
            if(d.status==='qr'){ showQR(d.qr); showStatus(status, '✅ Bot '+(d.id||'')+' — Escanea el QR', 'success'); }
            else if(d.status==='session_exists'){ hideQR(); setBotStatus('active','Sesión activa'); showStatus(status, '✅ Bot ya tiene sesión activa', 'success'); }
            else if(d.status==='pending_payment'){
                hideQR(); setBotStatus('inactive','Pago pendiente');
                showStatus(status, '⏳ ' + (d.message || 'Pago pendiente. Espera confirmación del administrador.'), 'info');
                if (d.id) { botID = d.id; $('botIdBadge').textContent = 'ID: ' + botID; window.botID = botID; }
                if (d.payment_status) { window.paymentStatus = d.payment_status; updatePaymentStatusMsg(d.payment_status); }
            }
            else if(d.status==='error'){ hideQR(); setBotStatus('error','Error'); showStatus(status, '❌ '+(d.message||'Error desconocido'), 'error'); }
            else { hideQR(); setBotStatus('error','Error'); showStatus(status, '❌ Respuesta inesperada', 'error'); }
        }).catch(err=>{ hideQR(); setBotStatus('error','Error'); showStatus(status, '❌ Error de red: '+err.message, 'error'); });
    });

    // Refrescar estado
    $('refreshStatusBtn').addEventListener('click', function(){
        const status=$('status'); hideStatus(status); showStatus(status, '⟳ Actualizando…', 'info');
        fetch('/bot/'+botID+'/status', {method:'GET'})
        .then(r=>r.json()).then(d=>{
            if(d.status==='active'){ setBotStatus('active','Sesión activa'); hideQR(); showStatus(status, '✅ Bot activo', 'success'); }
            else if(d.status==='qr'){ if(d.qr) showQR(d.qr); else setBotStatus('qr','QR pendiente'); showStatus(status, '📲 Escanea el QR', 'info'); }
            else if(d.status==='inactive'){ setBotStatus('inactive','Inactivo'); hideQR(); showStatus(status, '⏸️ Bot inactivo. Presiona "Iniciar Bot"', 'info'); }
            else if(d.status==='pending_payment'){ setBotStatus('inactive','Pago pendiente'); hideQR(); showStatus(status, '⏳ Pago pendiente', 'info'); }
            else { setBotStatus('error','Desconocido'); hideQR(); showStatus(status, '⚠️ Estado desconocido', 'error'); }
        }).catch(err=>{ setBotStatus('error','Error'); hideQR(); showStatus(status, '❌ Error al obtener estado: '+err.message, 'error'); });
    });

    // Actualizar Prompt
    $('updatePromptBtn').addEventListener('click', function(){
        const prompt=$('promptInput').value.trim(), status=$('promptStatus');
        hideStatus(status);
        if(!prompt){ showStatus(status, '❌ El prompt no puede estar vacío', 'error'); return; }
        showStatus(status, '⏳ Guardando…', 'info');
        fetch('/bot/'+botID+'/prompt', {method:'PUT', headers:{'Content-Type':'application/json'}, body:JSON.stringify({prompt:prompt})})
        .then(r=>r.json()).then(d=>{
            if(d.status==='ok'||d.status==='success') showStatus(status, '✅ Prompt actualizado correctamente', 'success');
            else showStatus(status, '❌ '+(d.message||d.error||'Error al guardar'), 'error');
        }).catch(()=>{ showStatus(status, '❌ Error de red', 'error'); });
    });

    // Actualizar Teléfono
    $('updatePhoneBtn').addEventListener('click', function(){
        const phone = $('newPhone').value.trim();
        const status = $('phoneStatus');
        hideStatus(status);
        if (!phone) {
            showStatus(status, '❌ Ingresa un número de teléfono', 'error');
            return;
        }
        if (phone.length < 8) {
            showStatus(status, '❌ El número debe tener al menos 8 caracteres', 'error');
            return;
        }
        showStatus(status, '⏳ Actualizando...', 'info');
        fetch('/user/phone', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ phone: phone })
        })
        .then(r => r.json())
        .then(d => {
            if (d.status === 'ok' || d.status === 'success') {
                showStatus(status, '✅ Teléfono actualizado correctamente', 'success');
                $('infoPhone').textContent = phone;
                $('newPhone').value = '';
                window.userPhone = phone; // Actualizar variable global
            } else {
                showStatus(status, '❌ ' + (d.error || d.message || 'Error al actualizar'), 'error');
            }
        })
        .catch(() => {
            showStatus(status, '❌ Error de red', 'error');
        });
    });

    // Cambiar Contraseña
    $('changePassBtn').addEventListener('click', function(){
        const pass=$('newPass').value.trim(), status=$('passStatus');
        hideStatus(status);
        if(!pass){ showStatus(status, '❌ Ingresa una contraseña', 'error'); return; }
        if(pass.length<6){ showStatus(status, '❌ Mínimo 6 caracteres', 'error'); return; }
        showStatus(status, '⏳ Actualizando…', 'info');
        fetch('/user/password', {method:'PUT', headers:{'Content-Type':'application/json'}, body:JSON.stringify({password:pass})})
        .then(r=>r.json()).then(d=>{
            if(d.status==='ok'||d.status==='success'){ showStatus(status, '✅ Contraseña actualizada', 'success'); $('newPass').value=''; }
            else showStatus(status, '❌ '+(d.error||d.message||'Error'), 'error');
        }).catch(()=>{ showStatus(status, '❌ Error de red', 'error'); });
    });

    // Cerrar sesión
    $('logoutBtn').addEventListener('click', function(){ fetch('/logout', {method:'POST'}).then(()=>window.location.href='/').catch(()=>window.location.href='/'); });

    setTimeout(function(){ const btn=$('refreshStatusBtn'); if(btn) btn.click(); }, 300);

    // Enter key support
    $('newPass').addEventListener('keydown', function(e){ if(e.key==='Enter'){ e.preventDefault(); $('changePassBtn').click(); } });
    $('newPhone').addEventListener('keydown', function(e){ if(e.key==='Enter'){ e.preventDefault(); $('updatePhoneBtn').click(); } });
    $('promptInput').addEventListener('keydown', function(e){ if(e.key==='Enter' && e.ctrlKey){ e.preventDefault(); $('updatePromptBtn').click(); } });

    window.botID = botID;

    // Navegación sidebar
    document.querySelectorAll('.sidebar-item').forEach(item => {
        item.addEventListener('click', function(e) {
            e.preventDefault();
            document.querySelectorAll('.sidebar-item').forEach(i => i.classList.remove('active'));
            this.classList.add('active');
            const targetId = this.getAttribute('href').substring(1);
            const target = document.getElementById(targetId);
            if (target) {
                target.scrollIntoView({ behavior: 'smooth', block: 'start' });
            }
        });
    });

    // Theme Toggle
    const themeToggleDash = document.getElementById('themeToggleDash');
    if (themeToggleDash) {
        const icon = themeToggleDash.querySelector('i');
        const currentTheme = localStorage.getItem('theme') || 'dark';
        if (currentTheme === 'light') {
            document.documentElement.classList.add('light-theme');
            icon.classList.replace('fa-moon', 'fa-sun');
        }
        themeToggleDash.addEventListener('click', () => {
            document.documentElement.classList.toggle('light-theme');
            if (document.documentElement.classList.contains('light-theme')) {
                localStorage.setItem('theme', 'light');
                icon.classList.replace('fa-moon', 'fa-sun');
            } else {
                localStorage.setItem('theme', 'dark');
                icon.classList.replace('fa-sun', 'fa-moon');
            }
        });
    }
})();
</script>
</body>
</html>`, role, user.Username, user.Email, user.Phone, role, botInfo, currentPrompt, botID, paymentStatus)

	c.Set("Content-Type", "text/html")
	return c.SendString(html)
})
