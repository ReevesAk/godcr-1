package page

import (
	"gioui.org/layout"

	"github.com/planetdecred/godcr/ui/decredmaterial"
	"github.com/planetdecred/godcr/ui/load"
	"github.com/planetdecred/godcr/ui/modal"
	"github.com/planetdecred/godcr/ui/page/components"
	"github.com/planetdecred/godcr/ui/values"
)

const DebugPageID = "Debug"

type debugItem struct {
	text   string
	page   string
	action func()
}

type DebugPage struct {
	*load.Load
	debugItems []debugItem
	list       *decredmaterial.ClickableList

	backButton decredmaterial.IconButton
}

func NewDebugPage(l *load.Load) *DebugPage {
	debugItems := []debugItem{
		{
			text: "Check wallet logs",
			page: LogPageID,
			action: func() {
				l.ChangeFragment(NewLogPage(l))
			},
		},
		{
			text: "Check statistics",
			page: StatisticsPageID,
			action: func() {
				l.ChangeFragment(NewStatPage(l))
			},
		},
	}

	pg := &DebugPage{
		Load:       l,
		debugItems: debugItems,
		list:       l.Theme.NewClickableList(layout.Vertical),
	}
	pg.list.Radius = decredmaterial.Radius(14)
	pg.list.IsShadowEnabled = true

	// TODO: temp removal till V1.0 is release
	// Add a "Reset DEX Client" option.
	// pg.debugItems = append(pg.debugItems, debugItem{
	// 	text: "Reset DEX Client",
	// 	action: func() {
	// 		pg.resetDexData()
	// 	},
	// })

	pg.backButton, _ = components.SubpageHeaderButtons(l)

	return pg
}

// ID is a unique string that identifies the page and may be used
// to differentiate this page from other pages.
// Part of the load.Page interface.
func (pg *DebugPage) ID() string {
	return DebugPageID
}

// OnNavigatedTo is called when the page is about to be displayed and
// may be used to initialize page features that are only relevant when
// the page is displayed.
// Part of the load.Page interface.
func (pg *DebugPage) OnNavigatedTo() {

}

// HandleUserInteractions is called just before Layout() to determine
// if any user interaction recently occurred on the page and may be
// used to update the page's UI components shortly before they are
// displayed.
// Part of the load.Page interface.
func (pg *DebugPage) HandleUserInteractions() {
	if clicked, item := pg.list.ItemClicked(); clicked {
		pg.debugItems[item].action()
	}
}

// OnNavigatedFrom is called when the page is about to be removed from
// the displayed window. This method should ideally be used to disable
// features that are irrelevant when the page is NOT displayed.
// NOTE: The page may be re-displayed on the app's window, in which case
// OnNavigatedTo() will be called again. This method should not destroy UI
// components unless they'll be recreated in the OnNavigatedTo() method.
// Part of the load.Page interface.
func (pg *DebugPage) OnNavigatedFrom() {}

func (pg *DebugPage) debugItem(gtx C, i int) D {
	return layout.Flex{}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.UniformInset(values.MarginPadding15).Layout(gtx, pg.Theme.Body1(pg.debugItems[i].text).Layout)
		}),
		layout.Flexed(1, func(gtx C) D {
			return layout.E.Layout(gtx, func(gtx C) D {
				return layout.UniformInset(values.MarginPadding15).Layout(gtx, func(gtx C) D {
					ic := decredmaterial.NewIcon(pg.Icons.ChevronRight)
					ic.Color = pg.Theme.Color.Gray1
					return ic.Layout(gtx, values.MarginPadding22)
				})
			})
		}),
	)
}

func (pg *DebugPage) layoutDebugItems(gtx C) {
	background := pg.Theme.Color.Surface
	card := pg.Theme.Card()
	card.Color = background
	card.Layout(gtx, func(gtx C) D {
		return pg.list.Layout(gtx, len(pg.debugItems), func(gtx C, i int) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return pg.debugItem(gtx, i)
				}),
				layout.Rigid(func(gtx C) D {
					if i == len(pg.debugItems)-1 {
						return layout.Dimensions{}
					}
					return layout.Inset{
						Left: values.MarginPadding16,
					}.Layout(gtx, pg.Theme.Separator().Layout)
				}),
			)
		})
	})
}

// Layout draws the page UI components into the provided layout context
// to be eventually drawn on screen.
// Part of the load.Page interface.
func (pg *DebugPage) Layout(gtx C) D {
	container := func(gtx C) D {
		sp := components.SubPage{
			Load:       pg.Load,
			Title:      "Debug",
			BackButton: pg.backButton,
			Back: func() {
				pg.PopFragment()
			},
			Body: func(gtx C) D {
				pg.layoutDebugItems(gtx)
				return layout.Dimensions{Size: gtx.Constraints.Max}
			},
		}
		return sp.Layout(gtx)

	}
	return components.UniformPadding(gtx, container)
}

func (pg *DebugPage) resetDexData() {
	// Show confirm modal before resetting dex client data.
	confirmModal := modal.NewInfoModal(pg.Load).
		Title("Confirm DEX Client Reset").
		Body("You may need to restart godcr before you can use the DEX again. Proceed?").
		NegativeButton(values.String(values.StrCancel), func() {}).
		PositiveButton("Reset DEX Client", func() {
			if pg.Dexc().Reset() {
				pg.Toast.Notify("DEX client data reset complete.")
			} else {
				pg.Toast.NotifyError("DEX client data reset failed. Check the logs.")
			}
		})
	pg.ShowModal(confirmModal)
}
