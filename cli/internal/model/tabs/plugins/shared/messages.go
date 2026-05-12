package shared

import "k8s-manager/cli/internal/market"

// Msg - интерфейс для сообщений, принадлежащих вкладке Plugins
type Msg interface {
	PluginMsg()
}

// OpenDetailMsg запрос на открытие детального окна плагина
type OpenDetailMsg struct {
	Summary market.PluginSummary
}

func (OpenDetailMsg) PluginMsg() {}

// BackToListMsg запрос на возврат с детального экрана на список
type BackToListMsg struct{}

func (BackToListMsg) PluginMsg() {}

// LibraryChangedMsg сигнализирует, что пользователь добавил/убрал плагин из библиотеки.
// Список должен подсветить новый статус "Installed"
type LibraryChangedMsg struct {
	PluginID  int64
	Installed bool
}

func (LibraryChangedMsg) PluginMsg() {}

// OpenWizardMsg испускается списком при нажатии Create. Роутер
// открывает wizard поверх текущего экрана
type OpenWizardMsg struct{}

func (OpenWizardMsg) PluginMsg() {}
