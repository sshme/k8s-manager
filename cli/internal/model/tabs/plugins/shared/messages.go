package shared

import "k8s-manager/cli/internal/market"

// OpenDetailMsg запрос на открытие детального окна плагина
type OpenDetailMsg struct {
	Summary market.PluginSummary
}

// BackToListMsg запрос на возврат с детального экрана на список
type BackToListMsg struct{}

// LibraryChangedMsg сигнализирует, что пользователь добавил/убрал плагин из библиотеки.
// Список должен подсветить новый статус "Installed"
type LibraryChangedMsg struct {
	PluginID  int64
	Installed bool
}

// OpenWizardMsg испускается списком при нажатии Create. Роутер
// открывает wizard поверх текущего экрана
type OpenWizardMsg struct{}
