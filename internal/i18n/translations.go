package i18n

var CurrentLanguage = "English"

var Dictionary = map[string]map[string]string{
	"English": {
		"spar":           "Spar",
		"duel":           "Duel",
		"team_building":  "Team Building",
		"profile":        "Profile",
		"options":        "Options",
		"debugger":       "Debugger",
		"debugger_title": "Debugger: Render/Control Tests",
		"quit":           "Quit",
		"back":           "Back",
		"lang_menu":      "L",
	},
	"Español": {
		"spar":           "Entrenar",
		"duel":           "Duelo",
		"team_building":  "Formar Equipo",
		"profile":        "Perfil",
		"options":        "Opciones",
		"debugger":       "Depurador",
		"debugger_title": "Depurador: Pruebas de Renderizado/Control",
		"quit":           "Salir",
		"back":           "Atrás",
		"lang_menu":      "L",
	},
}

func Get(key string) string {
	if langMap, ok := Dictionary[CurrentLanguage]; ok {
		if val, ok := langMap[key]; ok {
			return val
		}
	}
	return key
}
