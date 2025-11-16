package domain

// Структура команды
type Team struct {
	Name    string
	Members []TeamMember
}

// Структура участника команды
type TeamMember struct {
	ID       string
	Username string
	IsActive bool
}
