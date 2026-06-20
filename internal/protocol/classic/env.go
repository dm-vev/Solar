// env.go sends CPE environment packets based on level.Env settings.
//
// sendEnv is called after level streaming (in sendLevelFrom) and when
// environment properties are changed via /map command. It sends:
//   - Env colors (sky, cloud, fog, ambient, diffuse) — only if Set=true
//   - Weather type (sunny, rain, snow)
//   - Map properties (edge level, sides level, clouds height, max fog,
//     clouds speed, weather speed, weather fade, exp fog, skybox speeds)
//   - Lighting mode (classic or fancy)
//
// Each packet is only sent if the client supports the corresponding
// CPE extension (EnvColors, EnvWeatherType, EnvMapAspect, LightingMode).

package classic

import "github.com/solar-mc/solar/internal/world"

// sendEnv sends CPE environment packets based on level.Env settings.
// Only sends packets for extensions the client supports, and only for
// properties that are explicitly set (EnvColor.Set, non-default values).
func (s *session) sendEnv(env world.Env) {
	// Env colors (0=sky, 1=cloud, 2=fog, 3=ambient, 4=diffuse)
	for i, c := range env.Colors {
		if c.Set && s.supportsExt(cpeExtEnvColors) {
			_ = s.writePacket(encodeEnvColor(byte(i), int16(c.R), int16(c.G), int16(c.B)))
		}
	}

	// Weather
	if env.Weather != 0 && s.supportsExt(cpeExtEnvWeatherType) {
		_ = s.writePacket(encodeEnvWeatherType(env.Weather))
	}

	// Map appearance / properties
	if s.supportsExt(cpeExtEnvMapAspect) {
		if env.EdgeLevel >= 0 {
			_ = s.writePacket(encodeSetMapEnvProperty(0, int32(env.EdgeLevel)))
		}
		if env.SidesLevel >= 0 {
			_ = s.writePacket(encodeSetMapEnvProperty(1, int32(env.SidesLevel)))
		}
		if env.CloudsLevel >= 0 {
			_ = s.writePacket(encodeSetMapEnvProperty(2, int32(env.CloudsLevel)))
		}
		if env.MaxFog >= 0 {
			_ = s.writePacket(encodeSetMapEnvProperty(3, int32(env.MaxFog)))
		}
		if env.CloudsSpeed != 0 {
			_ = s.writePacket(encodeSetMapEnvProperty(4, env.CloudsSpeed))
		}
		if env.WeatherSpeed != 0 {
			_ = s.writePacket(encodeSetMapEnvProperty(5, env.WeatherSpeed))
		}
		if env.WeatherFade != 0 {
			_ = s.writePacket(encodeSetMapEnvProperty(6, env.WeatherFade))
		}
		if env.ExpFog {
			_ = s.writePacket(encodeSetMapEnvProperty(7, 1))
		}
		if env.SkyboxHorSpeed != 0 {
			_ = s.writePacket(encodeSetMapEnvProperty(8, env.SkyboxHorSpeed))
		}
		if env.SkyboxVerSpeed != 0 {
			_ = s.writePacket(encodeSetMapEnvProperty(9, env.SkyboxVerSpeed))
		}
	}

	// Lighting mode
	if env.LightingMode != 0 && s.supportsExt(cpeExtLightingMode) {
		_ = s.writePacket(encodeLightingMode(env.LightingMode, env.LightingLock))
	}
}
