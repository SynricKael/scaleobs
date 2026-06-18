use std::io::{Read, Write};
use std::net::TcpStream;
use std::process::{Child, Command};
use std::sync::Mutex;
use std::sync::atomic::{AtomicBool, Ordering};
use std::time::Duration;

use tauri::{
    AppHandle,
    menu::{Menu, MenuItem},
    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
    Manager, Url,
};

struct GatewayProcess(Mutex<Option<Child>>);
struct GatewayRunning(AtomicBool);

/// Open a service panel in the system browser.
/// The frontend passes panel_url (from services.yml) which is a direct URL.
/// For WebView2-unfriendly URLs (embedded auth, self-signed certs),
/// the system browser handles them correctly.
#[tauri::command]
fn open_panel(url: String, title: String) -> Result<(), String> {
    open::that(&url).map_err(|e| format!("Failed to open browser: {}", e))?;
    println!("[sop] opened in system browser '{}' -> {}", title, url);
    Ok(())
}

fn find_observer_root() -> Option<std::path::PathBuf> {
    let exe_path = std::env::current_exe().ok()?;
    Some(
        exe_path
            .parent()? // release/
            .parent()? // target/
            .parent()? // src-tauri/
            .parent()? // dashboard/
            .parent()? // Observer/
            .to_path_buf(),
    )
}

fn kill_stale_gateways() {
    let output = Command::new("taskkill")
        .args(&["/f", "/im", "gateway-release.exe"])
        .stdout(std::process::Stdio::null())
        .stderr(std::process::Stdio::null())
        .output();
    match output {
        Ok(o) if o.status.success() => println!("[sop] killed stale Gateway process(es)"),
        _ => {}
    }
}

fn check_gateway_health(port: u16) -> bool {
    let addr = format!("127.0.0.1:{}", port);
    if let Ok(mut stream) = TcpStream::connect_timeout(
        &addr.parse().unwrap(),
        Duration::from_millis(1500),
    ) {
        let _ = write!(stream, "GET /api/health HTTP/1.0\r\n\r\n");
        let mut buf = [0u8; 128];
        if let Ok(n) = stream.read(&mut buf) {
            let resp = String::from_utf8_lossy(&buf[..n]);
            return resp.contains("200");
        }
    }
    false
}

fn wait_for_gateway(port: u16, max_retries: u32) -> bool {
    for i in 0..max_retries {
        if check_gateway_health(port) {
            println!("[sop] Gateway ready (attempt {})", i + 1);
            return true;
        }
        if i < max_retries - 1 {
            println!("[sop] waiting for Gateway... ({}/{})", i + 1, max_retries);
            std::thread::sleep(Duration::from_millis(1000));
        }
    }
    eprintln!("[sop] Gateway did not start within timeout");
    false
}

fn spawn_gateway_process() -> Option<Child> {
    let observer_root = find_observer_root()?;
    let gateway_exe = observer_root.join("gateway-release.exe");
    let config_path = observer_root.join("config/services.yml");

    if !gateway_exe.exists() {
        eprintln!("[sop] gateway-release.exe not found at {:?}", gateway_exe);
        return None;
    }
    if !config_path.exists() {
        eprintln!("[sop] config not found at {:?}", config_path);
        return None;
    }

    kill_stale_gateways();
    std::thread::sleep(Duration::from_millis(800));

    println!("[sop] starting Gateway: {:?}", gateway_exe);

    match Command::new(&gateway_exe)
        .env("CONFIG_PATH", config_path.to_string_lossy().as_ref())
        .env("JWT_SECRET", "sop-secret-key-2024")
        .env("ADMIN_USERNAME", "admin")
        .env("ADMIN_PASSWORD", "admin123")
        .env("AGENT_TOKEN", "sop-agent-token-2024")
        .stdout(std::process::Stdio::null())
        .stderr(std::process::Stdio::null())
        .spawn()
    {
        Ok(mut c) => {
            println!("[sop] Gateway spawned (PID: {})", c.id());
            if wait_for_gateway(8080, 8) {
                println!("[sop] Gateway is ready");
                Some(c)
            } else {
                let _ = c.kill();
                let _ = c.wait();
                None
            }
        }
        Err(e) => {
            eprintln!("[sop] failed to start Gateway: {}", e);
            None
        }
    }
}

fn start_health_watcher(app: AppHandle) {
    // Spawn a background thread that checks Gateway health every 15s
    // If 3 consecutive checks fail, auto-restart the Gateway.
    std::thread::spawn(move || {
        let mut consecutive_failures = 0;
        const MAX_FAILURES: u32 = 3;
        let check_interval = Duration::from_secs(15);

        loop {
            std::thread::sleep(check_interval);

            let is_healthy = check_gateway_health(8080);

            // Update the running flag
            let state = app.state::<GatewayRunning>();
            state.0.store(is_healthy, Ordering::SeqCst);

            if is_healthy {
                consecutive_failures = 0;
                continue;
            }

            consecutive_failures += 1;
            println!(
                "[sop] health check failed ({}/{})",
                consecutive_failures, MAX_FAILURES
            );

            if consecutive_failures >= MAX_FAILURES {
                println!("[sop] Gateway seems dead, restarting...");
                consecutive_failures = 0;

                let new_child = spawn_gateway_process();
                let gw_state = app.state::<GatewayProcess>();
                let is_alive = {
                    let mut guard = gw_state.0.lock().unwrap_or_else(|e| e.into_inner());
                    if let Some(ref mut old) = *guard {
                        let _ = old.kill();
                        let _ = old.wait();
                    }
                    *guard = new_child;
                    guard.is_some()
                };
                if is_alive {
                    println!("[sop] Gateway auto-restarted successfully");
                    state.0.store(true, Ordering::SeqCst);
                } else {
                    eprintln!("[sop] Gateway auto-restart failed");
                }
            }
        }
    });
}

fn kill_gateway(state: &tauri::State<GatewayProcess>) {
    if let Ok(mut guard) = state.0.lock() {
        if let Some(ref mut child) = *guard {
            println!("[sop] stopping Gateway (PID: {})", child.id());
            let _ = child.kill();
            let _ = child.wait();
            println!("[sop] Gateway stopped");
        }
        *guard = None;
    }
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .manage(GatewayProcess(Mutex::new(None)))
        .manage(GatewayRunning(AtomicBool::new(false)))
        .setup(|app| {
            // Start Gateway
            let gateway_child = spawn_gateway_process();
            if gateway_child.is_some() {
                println!("[sop] Gateway running in background");
                let state = app.state::<GatewayRunning>();
                state.0.store(true, Ordering::SeqCst);

                // Navigate to the Gateway-served dashboard
                if let Some(window) = app.get_webview_window("main") {
                    let url = Url::parse("http://localhost:8080/").unwrap();
                    let _ = window.navigate(url);
                    println!("[sop] Navigated to Gateway dashboard");
                }
            } else {
                eprintln!("[sop] Gateway not available — frontend will show offline state");
            }
            let proc_state = app.state::<GatewayProcess>();
            if let Ok(mut guard) = proc_state.0.lock() {
                *guard = gateway_child;
            }

            // Start background health watcher (15s interval, auto-restart on 3 failures)
            start_health_watcher(app.handle().clone());

            // Build system tray menu
            let open_item =
                MenuItem::with_id(app, "open", "打开主界面", true, None::<&str>)?;
            let restart_item =
                MenuItem::with_id(app, "restart", "重启后端服务", true, None::<&str>)?;
            let separator1 = tauri::menu::PredefinedMenuItem::separator(app)?;
            let frps_item =
                MenuItem::with_id(app, "frps", "FRP 服务端", true, None::<&str>)?;
            let headscale_item =
                MenuItem::with_id(app, "headscale", "Headscale 网络管理", true, None::<&str>)?;
            let separator2 = tauri::menu::PredefinedMenuItem::separator(app)?;
            let quit_item = MenuItem::with_id(app, "quit", "退出", true, None::<&str>)?;

            let menu = Menu::with_items(
                app,
                &[&open_item, &restart_item, &separator1, &frps_item, &headscale_item, &separator2, &quit_item],
            )?;

            // Build tray icon
            let _tray = TrayIconBuilder::new()
                .tooltip("服务器运维平台")
                .icon(app.default_window_icon().unwrap().clone())
                .menu(&menu)
                .on_menu_event(|app, event| match event.id.as_ref() {
                    "open" => {
                        if let Some(window) = app.get_webview_window("main") {
                            let _ = window.show();
                            let _ = window.set_focus();
                        }
                    }
                    "restart" => {
                        // Kill existing Gateway, then start a new one
                        let state = app.state::<GatewayProcess>();
                        kill_gateway(&state);
                        let new_child = spawn_gateway_process();
                        let is_alive = new_child.is_some();
                        if let Ok(mut guard) = state.0.lock() {
                            *guard = new_child;
                        }
                        if is_alive {
                            let running = app.state::<GatewayRunning>();
                            running.0.store(true, Ordering::SeqCst);
                            println!("[sop] Gateway restarted via tray menu");
                        }
                    }
                    "frps" => {
                        // Use proxy URL — browsers strip embedded credentials for security
                        let _ = open::that("http://localhost:8080/p/frps/");
                    }
                    "headscale" => {
                        // Direct URL — browser lets user bypass self-signed cert warning once
                        let _ = open::that("https://8.135.39.171:8444/web/");
                    }
                    "quit" => {
                        let state = app.state::<GatewayProcess>();
                        kill_gateway(&state);
                        app.exit(0);
                    }
                    _ => {}
                })
                .on_tray_icon_event(|tray, event| {
                    if let TrayIconEvent::Click {
                        button: MouseButton::Left,
                        button_state: MouseButtonState::Up,
                        ..
                    } = event
                    {
                        let app = tray.app_handle();
                        if let Some(window) = app.get_webview_window("main") {
                            let _ = window.show();
                            let _ = window.set_focus();
                        }
                    }
                })
                .build(app)?;

            println!("[sop] Tray icon initialized");

            Ok(())
        })
        .invoke_handler(tauri::generate_handler![open_panel])
        .build(tauri::generate_context!())
        .expect("error while building tauri application")
        .run(|app_handle, event| {
            if let tauri::RunEvent::WindowEvent {
                label,
                event: window_event,
                ..
            } = event
            {
                if label == "main" {
                    if let tauri::WindowEvent::CloseRequested { api, .. } = window_event {
                        if let Some(window) = app_handle.get_webview_window("main") {
                            let _ = window.hide();
                        }
                        api.prevent_close();
                    }
                }
            }
        });
}
