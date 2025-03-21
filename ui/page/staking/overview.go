package staking

import (
	"context"
	"fmt"

	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/widget"

	"github.com/decred/dcrd/dcrutil/v4"
	"github.com/planetdecred/dcrlibwallet"
	"github.com/planetdecred/godcr/ui/decredmaterial"
	"github.com/planetdecred/godcr/ui/load"
	"github.com/planetdecred/godcr/ui/modal"
	"github.com/planetdecred/godcr/ui/page/components"
	"github.com/planetdecred/godcr/ui/page/overview"
	tpage "github.com/planetdecred/godcr/ui/page/transaction"
	"github.com/planetdecred/godcr/ui/values"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

const OverviewPageID = "staking"

type Page struct {
	*load.Load
	list *widget.List

	ctx       context.Context // page context
	ctxCancel context.CancelFunc

	ticketBuyerWallet *dcrlibwallet.Wallet
	ticketsLive       *decredmaterial.ClickableList

	autoPurchaseSettings *decredmaterial.Clickable
	autoPurchase         *decredmaterial.Switch

	stakeBtn  decredmaterial.Button
	toTickets decredmaterial.TextAndIconButton

	ticketOverview *dcrlibwallet.StakingOverview
	liveTickets    []*transactionItem

	ticketPrice  string
	totalRewards string
}

func NewStakingPage(l *load.Load) *Page {
	pg := &Page{
		Load: l,
	}

	pg.list = &widget.List{
		List: layout.List{
			Axis: layout.Vertical,
		},
	}

	pg.ticketOverview = new(dcrlibwallet.StakingOverview)

	pg.initStakePriceWidget()
	pg.initLiveStakeWidget()
	pg.loadPageData()

	return pg
}

// ID is a unique string that identifies the page and may be used
// to differentiate this page from other pages.
// Part of the load.Page interface.
func (pg *Page) ID() string {
	return OverviewPageID
}

// OnNavigatedTo is called when the page is about to be displayed and
// may be used to initialize page features that are only relevant when
// the page is displayed.
// Part of the load.Page interface.
func (pg *Page) OnNavigatedTo() {
	// pg.ctx is used to load known vsps in background and
	// canceled in OnNavigatedFrom().
	pg.ctx, pg.ctxCancel = context.WithCancel(context.TODO())

	pg.setTBWallet()
	ticketPrice, err := pg.WL.MultiWallet.TicketPrice()
	if err != nil {
		pg.ticketPrice = "0 DCR"
		bestBlock := pg.WL.MultiWallet.GetBestBlock()
		activationHeight := pg.WL.MultiWallet.DCP0001ActivationBlockHeight()
		if bestBlock.Height < activationHeight {
			modal.NewInfoModal(pg.Load).
				Title("Staking notification").
				SetupWithTemplate(modal.TicketPriceErrorTemplate).
				SetContentAlignment(layout.Center, layout.Center).
				NegativeButton(values.String(values.StrCancel), func() {}).
				PositiveButton("Go to Overview", func() {
					pg.ChangeFragment(overview.NewOverviewPage(pg.Load))
				}).
				Show()
		} else {
			pg.Toast.NotifyError("Unable to fetch ticket price: " + err.Error())
		}
	} else {
		pg.ticketPrice = dcrutil.Amount(ticketPrice.TicketPrice).String()
	}

	pg.loadPageData() // starts go routines to refresh the display which is just about to be displayed, ok?

	pg.autoPurchase.SetChecked(pg.ticketBuyerWallet.IsAutoTicketsPurchaseActive())
}

func (pg *Page) setTBWallet() {
	for _, wal := range pg.WL.SortedWalletList() {
		if wal.TicketBuyerConfigIsSet() {
			pg.ticketBuyerWallet = wal
		}
	}

	// if there are no tickets with config set, select the first wallet.
	if pg.ticketBuyerWallet == nil {
		pg.ticketBuyerWallet = pg.WL.SortedWalletList()[0]
	}
}

func (pg *Page) loadPageData() {
	go func() {
		if len(pg.WL.MultiWallet.KnownVSPs()) == 0 {
			// TODO: Does this page need this list?
			if pg.ctx != nil {
				pg.WL.MultiWallet.ReloadVSPList(pg.ctx)
			}
		}

		totalRewards, err := pg.WL.MultiWallet.TotalStakingRewards()
		if err != nil {
			pg.Toast.NotifyError(err.Error())
		} else {
			pg.totalRewards = dcrutil.Amount(totalRewards).String()
		}

		overview, err := pg.WL.MultiWallet.StakingOverview()
		if err != nil {
			pg.Toast.NotifyError(err.Error())
		} else {
			pg.ticketOverview = overview
		}

		pg.RefreshWindow()
	}()

	go func() {
		mw := pg.WL.MultiWallet
		tickets, err := allLiveTickets(mw)
		if err != nil {
			pg.Toast.NotifyError(err.Error())
			return
		}

		txItems, err := stakeToTransactionItems(pg.Load, tickets, true, func(filter int32) bool {
			switch filter {
			case dcrlibwallet.TxFilterUnmined:
				fallthrough
			case dcrlibwallet.TxFilterImmature:
				fallthrough
			case dcrlibwallet.TxFilterLive:
				return true
			}

			return false
		})
		if err != nil {
			pg.Toast.NotifyError(err.Error())
			return
		}

		pg.liveTickets = txItems
		pg.RefreshWindow()
	}()
}

// Layout draws the page UI components into the provided layout context
// to be eventually drawn on screen.
// Part of the load.Page interface.
func (pg *Page) Layout(gtx C) D {
	widgets := []layout.Widget{
		func(gtx C) D {
			return components.UniformHorizontalPadding(gtx, pg.stakePriceSection)
		},
		func(gtx C) D {
			return components.UniformHorizontalPadding(gtx, pg.walletBalanceLayout)
		},
		func(gtx C) D {
			return components.UniformHorizontalPadding(gtx, pg.stakeLiveSection)
		},
		func(gtx C) D {
			return components.UniformHorizontalPadding(gtx, pg.stakingRecordSection)
		},
	}

	return layout.Inset{Top: values.MarginPadding24}.Layout(gtx, func(gtx C) D {
		return pg.Theme.List(pg.list).Layout(gtx, len(widgets), func(gtx C, i int) D {
			return widgets[i](gtx)
		})
	})
}

func (pg *Page) pageSections(gtx C, body layout.Widget) D {
	return layout.Inset{
		Bottom: values.MarginPadding8,
	}.Layout(gtx, func(gtx C) D {
		return pg.Theme.Card().Layout(gtx, func(gtx C) D {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.UniformInset(values.MarginPadding16).Layout(gtx, body)
		})
	})
}

func (pg *Page) titleRow(gtx C, leftWidget, rightWidget func(C) D) D {
	return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceBetween}.Layout(gtx,
		layout.Rigid(leftWidget),
		layout.Rigid(rightWidget),
	)
}

// HandleUserInteractions is called just before Layout() to determine
// if any user interaction recently occurred on the page and may be
// used to update the page's UI components shortly before they are
// displayed.
// Part of the load.Page interface.
func (pg *Page) HandleUserInteractions() {
	if pg.stakeBtn.Clicked() {
		newStakingModal(pg.Load).
			TicketPurchased(func() {
				align := layout.Center
				successIcon := decredmaterial.NewIcon(pg.Icons.ActionCheckCircle)
				successIcon.Color = pg.Theme.Color.Success
				info := modal.NewInfoModal(pg.Load).
					Icon(successIcon).
					Title("Ticket(s) Confirmed").
					SetContentAlignment(align, align).
					PositiveButton("Back to staking", func() {})
				pg.ShowModal(info)
				pg.loadPageData()
			}).Show()
	}

	if pg.toTickets.Button.Clicked() {
		pg.ChangeFragment(newListPage(pg.Load))
	}

	if clicked, selectedItem := pg.ticketsLive.ItemClicked(); clicked {
		pg.ChangeFragment(tpage.NewTransactionDetailsPage(pg.Load, pg.liveTickets[selectedItem].transaction))
	}

	if pg.autoPurchase.Changed() {
		if pg.autoPurchase.IsChecked() {
			if pg.ticketBuyerWallet.TicketBuyerConfigIsSet() {
				pg.startTicketBuyerPasswordModal()
			} else {
				newTicketBuyerModal(pg.Load).
					OnCancel(func() {
						pg.autoPurchase.SetChecked(false)
					}).
					OnSettingsSaved(func() {
						pg.startTicketBuyerPasswordModal()
						pg.Toast.Notify("Auto ticket purchase setting saved successfully.")
					}).
					Show()
			}
		} else {
			pg.WL.MultiWallet.StopAutoTicketsPurchase(pg.ticketBuyerWallet.ID)
		}
	}

	if pg.autoPurchaseSettings.Clicked() {
		if pg.ticketBuyerWallet.IsAutoTicketsPurchaseActive() {
			pg.Toast.NotifyError("Settings can not be modified when ticket buyer is running.")
			return
		}

		pg.ticketBuyerSettingsModal()
	}
}

func (pg *Page) ticketBuyerSettingsModal() {
	newTicketBuyerModal(pg.Load).
		OnSettingsSaved(func() {
			pg.Toast.Notify("Auto ticket purchase setting saved successfully.")
			pg.setTBWallet()
		}).
		OnCancel(func() {
			pg.autoPurchase.SetChecked(false)
		}).
		Show()
}

func (pg *Page) startTicketBuyerPasswordModal() {
	tbConfig := pg.ticketBuyerWallet.AutoTicketsBuyerConfig()
	balToMaintain := dcrlibwallet.AmountCoin(tbConfig.BalanceToMaintain)
	name, err := pg.ticketBuyerWallet.AccountNameRaw(uint32(tbConfig.PurchaseAccount))
	if err != nil {
		pg.Toast.NotifyError("Ticket buyer acount error: " + err.Error())
		return
	}

	modal.NewPasswordModal(pg.Load).
		Title("Confirm Automatic Ticket Purchase").
		SetCancelable(false).
		UseCustomWidget(func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(pg.Theme.Label(values.TextSize14, fmt.Sprintf("Wallet to purchase from: %s", pg.ticketBuyerWallet.Name)).Layout),
				layout.Rigid(pg.Theme.Label(values.TextSize14, fmt.Sprintf("Selected account: %s", name)).Layout),
				layout.Rigid(pg.Theme.Label(values.TextSize14, fmt.Sprintf("Balance to maintain: %2.f", balToMaintain)).Layout), layout.Rigid(func(gtx C) D {
					label := pg.Theme.Label(values.TextSize14, fmt.Sprintf("VSP: %s", tbConfig.VspHost))
					return layout.Inset{Bottom: values.MarginPadding12}.Layout(gtx, label.Layout)
				}),
				layout.Rigid(func(gtx C) D {
					return decredmaterial.LinearLayout{
						Width:      decredmaterial.MatchParent,
						Height:     decredmaterial.WrapContent,
						Background: pg.Theme.Color.LightBlue,
						Padding: layout.Inset{
							Top:    values.MarginPadding12,
							Bottom: values.MarginPadding12,
						},
						Border:    decredmaterial.Border{Radius: decredmaterial.Radius(8)},
						Direction: layout.Center,
						Alignment: layout.Middle,
					}.Layout2(gtx, func(gtx C) D {
						return layout.Inset{Bottom: values.MarginPadding4}.Layout(gtx, func(gtx C) D {
							msg := "Godcr must remain running, for tickets to be automatically purchased"
							txt := pg.Theme.Label(values.TextSize14, msg)
							txt.Alignment = text.Middle
							txt.Color = pg.Theme.Color.GrayText3
							if pg.WL.MultiWallet.ReadBoolConfigValueForKey(load.DarkModeConfigKey, false) {
								txt.Color = pg.Theme.Color.Gray3
							}
							return txt.Layout(gtx)
						})
					})
				}),
			)
		}).
		NegativeButton("Cancel", func() {
			pg.autoPurchase.SetChecked(false)
		}).
		PositiveButton("Confirm", func(password string, pm *modal.PasswordModal) bool {
			if !pg.WL.MultiWallet.IsConnectedToDecredNetwork() {
				pg.Toast.NotifyError(values.String(values.StrNotConnected))
				pm.SetLoading(false)
				pg.autoPurchase.SetChecked(false)
				return false
			}

			go func() {
				err := pg.ticketBuyerWallet.StartTicketBuyer([]byte(password))
				if err != nil {
					pg.Toast.NotifyError(err.Error())
					pm.SetLoading(false)
					return
				}

				pg.autoPurchase.SetChecked(pg.ticketBuyerWallet.IsAutoTicketsPurchaseActive())
				pg.RefreshWindow()
			}()
			pm.Dismiss()

			return false
		}).Show()
}

// OnNavigatedFrom is called when the page is about to be removed from
// the displayed window. This method should ideally be used to disable
// features that are irrelevant when the page is NOT displayed.
// NOTE: The page may be re-displayed on the app's window, in which case
// OnNavigatedTo() will be called again. This method should not destroy UI
// components unless they'll be recreated in the OnNavigatedTo() method.
// Part of the load.Page interface.
func (pg *Page) OnNavigatedFrom() {
	pg.ctxCancel()
}
