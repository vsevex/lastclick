import { isTMA, mockTelegramEnv, emitEvent } from "@telegram-apps/sdk";

export async function ensureTelegramEnv(): Promise<void> {
  if (process.env.NODE_ENV !== "development") return;

  const isInTelegram = await isTMA("complete");
  if (isInTelegram) return;

  const themeParams: Record<string, `#${string}`> = {
    bg_color: "#0B0F14",
    text_color: "#ffffff",
    hint_color: "#8b9cc4",
    link_color: "#6ab3f3",
    button_color: "#6c93d6",
    button_text_color: "#ffffff",
    secondary_bg_color: "#131820",
    header_bg_color: "#0B0F14",
    accent_text_color: "#6ab3f3",
    section_bg_color: "#131820",
    section_header_text_color: "#6ab3f3",
    subtitle_text_color: "#8b9cc4",
    destructive_text_color: "#ef5b5b",
  };

  const initDataRaw = new URLSearchParams([
    ["auth_date", String(Math.floor(Date.now() / 1000))],
    ["hash", "0".repeat(64)],
    ["signature", "0".repeat(64)],
    [
      "user",
      JSON.stringify({
        id: 1,
        first_name: "Dev",
        last_name: "",
        username: "dev",
        language_code: "en",
        is_premium: false,
        allows_write_to_pm: true,
      }),
    ],
  ]).toString();

  const launchParams = {
    tgWebAppThemeParams: themeParams,
    tgWebAppData: initDataRaw,
    tgWebAppVersion: "8.0",
    tgWebAppPlatform: "tdesktop",
  };

  mockTelegramEnv({
    onEvent(event) {
      if (event[0] === "web_app_request_theme") {
        emitEvent("theme_changed", { theme_params: themeParams });
      }
      if (event[0] === "web_app_request_viewport") {
        emitEvent("viewport_changed", {
          height: window.innerHeight,
          width: window.innerWidth,
          is_expanded: true,
          is_state_stable: true,
        });
      }
    },
    launchParams,
  });

  console.info("[TG] Mocked Telegram environment for development");
}
