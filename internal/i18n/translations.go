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
		"back_symbol":    "\uf060", // Nerd Font Back Arrow
		"lang_menu":      "\uf0ac", // Nerd Font Globe
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
		"back_symbol":    "\uf060", // Nerd Font Back Arrow
		"lang_menu":      "\uf0ac", // Nerd Font Globe
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
