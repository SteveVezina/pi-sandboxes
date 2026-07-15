#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_notification::init())
        .setup(|app| {
            build_tray(app)?;
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running PI Sandbox GUI");
}

fn build_tray(app: &mut tauri::App) -> tauri::Result<()> {
    use tauri::{
        menu::{Menu, MenuItem, PredefinedMenuItem},
        tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
        Emitter,
    };

    let status = MenuItem::with_id(app, "status", "PI Sandbox is running", false, None::<&str>)?;
    let dashboard = MenuItem::with_id(app, "dashboard", "Go to Dashboard", true, None::<&str>)?;
    let sessions = MenuItem::with_id(app, "sessions", "Sessions", true, None::<&str>)?;
    let settings = MenuItem::with_id(app, "settings", "Settings...", true, Some("CmdOrCtrl+,"))?;
    let diagnostics = MenuItem::with_id(app, "policies", "Policies and diagnostics", true, None::<&str>)?;
    let check_updates = MenuItem::with_id(app, "check_updates", "Check for updates", false, None::<&str>)?;
    let restart = MenuItem::with_id(app, "restart", "Restart", true, Some("CmdOrCtrl+R"))?;
    let quit = MenuItem::with_id(app, "quit", "Quit PI Sandbox", true, Some("CmdOrCtrl+Q"))?;
    let separator_one = PredefinedMenuItem::separator(app)?;
    let separator_two = PredefinedMenuItem::separator(app)?;
    let separator_three = PredefinedMenuItem::separator(app)?;

    let menu = Menu::with_items(
        app,
        &[
            &status,
            &separator_one,
            &dashboard,
            &sessions,
            &settings,
            &diagnostics,
            &separator_two,
            &check_updates,
            &restart,
            &separator_three,
            &quit,
        ],
    )?;

    let icon = app.default_window_icon().cloned();

    let mut tray = TrayIconBuilder::with_id("pi-sandbox")
        .tooltip("PI Sandbox")
        .menu(&menu)
        .show_menu_on_left_click(true)
        .on_menu_event(|app, event| match event.id().as_ref() {
            "dashboard" | "sessions" | "settings" | "policies" => {
                show_main_window(app);
                let _ = app.emit("pi://navigate", event.id().as_ref());
            }
            "restart" => app.restart(),
            "quit" => app.exit(0),
            _ => {}
        })
        .on_tray_icon_event(|tray, event| {
            if let TrayIconEvent::Click {
                button: MouseButton::Left,
                button_state: MouseButtonState::Up,
                ..
            } = event
            {
                show_main_window(tray.app_handle());
            }
        });

    if let Some(icon) = icon {
        tray = tray.icon(icon);
    }

    tray.build(app)?;
    Ok(())
}

fn show_main_window(app: &tauri::AppHandle) {
    use tauri::Manager;

    if let Some(window) = app.get_webview_window("main") {
        let _ = window.show();
        let _ = window.unminimize();
        let _ = window.set_focus();
    }
}
