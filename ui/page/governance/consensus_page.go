package governance

import (
	"context"
	"time"

	"gioui.org/font/gofont"
	"gioui.org/io/clipboard"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/planetdecred/dcrlibwallet"
	"github.com/planetdecred/godcr/ui/decredmaterial"
	"github.com/planetdecred/godcr/ui/load"
	"github.com/planetdecred/godcr/ui/modal"
	"github.com/planetdecred/godcr/ui/page/components"
	"github.com/planetdecred/godcr/ui/values"
)

const ConsensusPageID = "Consensus"

type ConsensusPage struct {
	*load.Load

	ctx       context.Context // page context
	ctxCancel context.CancelFunc

	multiWallet    *dcrlibwallet.MultiWallet
	wallets        []*dcrlibwallet.Wallet
	LiveTickets    []*dcrlibwallet.Transaction
	consensusItems []*components.ConsensusItem

	listContainer       *widget.List
	syncButton          *widget.Clickable
	viewVotingDashboard *decredmaterial.Clickable
	copyRedirectURL     *decredmaterial.Clickable
	redirectIcon        *decredmaterial.Image

	walletDropDown *decredmaterial.DropDown
	orderDropDown  *decredmaterial.DropDown
	consensusList  *decredmaterial.ClickableList

	searchEditor decredmaterial.Editor
	infoButton   decredmaterial.IconButton

	syncCompleted bool
	isSyncing     bool
}

func NewConsensusPage(l *load.Load) *ConsensusPage {
	pg := &ConsensusPage{
		Load:          l,
		multiWallet:   l.WL.MultiWallet,
		wallets:       l.WL.SortedWalletList(),
		consensusList: l.Theme.NewClickableList(layout.Vertical),
		listContainer: &widget.List{
			List: layout.List{Axis: layout.Vertical},
		},
		syncButton:          new(widget.Clickable),
		redirectIcon:        l.Icons.RedirectIcon,
		viewVotingDashboard: l.Theme.NewClickable(false),
		copyRedirectURL:     l.Theme.NewClickable(false),
	}

	pg.searchEditor = l.Theme.IconEditor(new(widget.Editor), "Search", l.Icons.SearchIcon, true)
	pg.searchEditor.Editor.SingleLine, pg.searchEditor.Editor.Submit, pg.searchEditor.Bordered = true, true, false

	_, pg.infoButton = components.SubpageHeaderButtons(l)
	pg.infoButton.Size = values.MarginPadding20

	pg.walletDropDown = components.CreateOrUpdateWalletDropDown(pg.Load, &pg.walletDropDown, pg.wallets, values.TxDropdownGroup, 0)
	pg.orderDropDown = components.CreateOrderDropDown(l, values.ConsensusDropdownGroup, 0)

	return pg
}

func (pg *ConsensusPage) ID() string {
	return ConsensusPageID
}

func (pg *ConsensusPage) OnNavigatedTo() {
	pg.ctx, pg.ctxCancel = context.WithCancel(context.TODO())
	pg.FetchAgendas()
}

func (pg *ConsensusPage) OnNavigatedFrom() {
	if pg.ctxCancel != nil {
		pg.ctxCancel()
	}
}

func (pg *ConsensusPage) HandleUserInteractions() {
	for pg.walletDropDown.Changed() {
		pg.FetchAgendas()
	}

	for pg.orderDropDown.Changed() {
		pg.FetchAgendas()
	}

	for i := range pg.consensusItems {
		if pg.consensusItems[i].VoteButton.Clicked() {
			newAgendaVoteModal(pg.Load, &pg.consensusItems[i].Agenda, func() {
				go pg.FetchAgendas() // re-fetch agendas when modal is dismissed
			}).Show()
		}
	}

	for pg.syncButton.Clicked() {
		go pg.FetchAgendas()
	}

	if pg.infoButton.Button.Clicked() {
		modal.NewInfoModal(pg.Load).
			Title("Consensus changes").
			Body("On-chain voting for upgrading the Decred network consensus rules.").
			SetCancelable(true).
			PositiveButton("Got it", func() {}).Show()
	}

	for pg.viewVotingDashboard.Clicked() {
		host := "https://voting.decred.org"
		if pg.WL.MultiWallet.NetType() == dcrlibwallet.Testnet3 {
			host = "https://voting.decred.org/testnet"
		}

		info := modal.NewInfoModal(pg.Load).
			Title("Consensus Vote Dashboard").
			Body("Copy and paste the link below in your browser, to the view consensus vote dashboard.").
			SetCancelable(true).
			UseCustomWidget(func(gtx C) D {
				return layout.Stack{}.Layout(gtx,
					layout.Stacked(func(gtx C) D {
						border := widget.Border{Color: pg.Theme.Color.Gray4, CornerRadius: values.MarginPadding10, Width: values.MarginPadding2}
						wrapper := pg.Theme.Card()
						wrapper.Color = pg.Theme.Color.Gray4
						return border.Layout(gtx, func(gtx C) D {
							return wrapper.Layout(gtx, func(gtx C) D {
								return layout.UniformInset(values.MarginPadding10).Layout(gtx, func(gtx C) D {
									return layout.Flex{}.Layout(gtx,
										layout.Flexed(0.9, pg.Theme.Body1(host).Layout),
										layout.Flexed(0.1, func(gtx C) D {
											return layout.E.Layout(gtx, func(gtx C) D {
												if pg.copyRedirectURL.Clicked() {
													clipboard.WriteOp{Text: host}.Add(gtx.Ops)
													pg.Toast.Notify("URL copied")
												}
												return pg.copyRedirectURL.Layout(gtx, pg.Icons.CopyIcon.Layout24dp)
											})
										}),
									)
								})
							})
						})
					}),
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{
							Top:  values.MarginPaddingMinus10,
							Left: values.MarginPadding10,
						}.Layout(gtx, func(gtx C) D {
							label := pg.Theme.Body2("Web URL")
							label.Color = pg.Theme.Color.GrayText2
							return label.Layout(gtx)
						})
					}),
				)
			}).
			PositiveButton("Got it", func() {})
		pg.ShowModal(info)
	}

	if pg.syncCompleted {
		time.AfterFunc(time.Second*1, func() {
			pg.syncCompleted = false
			pg.RefreshWindow()
		})
	}

	pg.searchEditor.EditorIconButtonEvent = func() {
		//TODO: consensus search functionality
	}
}

func (pg *ConsensusPage) FetchAgendas() {
	newestFirst := pg.orderDropDown.SelectedIndex() == 0
	selectedWallet := pg.wallets[pg.walletDropDown.SelectedIndex()]

	pg.isSyncing = true

	// Fetch (or re-fetch) agendas in background as this makes
	// a network call. Refresh the window once the call completes.
	go func() {
		pg.consensusItems = components.LoadAgendas(pg.Load, selectedWallet, newestFirst)
		pg.isSyncing = false
		pg.syncCompleted = true
		pg.RefreshWindow()
	}()

	// Refresh the window now to signify that the syncing
	// has started with pg.isSyncing set to true above.
	pg.RefreshWindow()
}

func (pg *ConsensusPage) Layout(gtx C) D {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(pg.Theme.Label(values.TextSize20, "Consensus Changes").Layout), // Do we really need to display the title? nav is proposals already
						layout.Rigid(pg.infoButton.Layout),
					)
				}),
				layout.Flexed(1, func(gtx C) D {
					return layout.E.Layout(gtx, pg.layoutRedirectVoting)
				}),
			)
		}),
		layout.Flexed(1, func(gtx C) D {
			return layout.Inset{Top: values.MarginPadding10}.Layout(gtx, func(gtx C) D {
				return layout.Stack{}.Layout(gtx,
					layout.Expanded(func(gtx C) D {
						return layout.Inset{
							Top: values.MarginPadding60,
						}.Layout(gtx, pg.layoutContent)
					}),
					layout.Expanded(func(gtx C) D {
						gtx.Constraints.Max.X = gtx.Px(values.MarginPadding150)
						gtx.Constraints.Min.X = gtx.Constraints.Max.X

						card := pg.Theme.Card()
						card.Radius = decredmaterial.Radius(8)
						return card.Layout(gtx, func(gtx C) D {
							return layout.Inset{
								Left:   values.MarginPadding10,
								Right:  values.MarginPadding10,
								Top:    values.MarginPadding2,
								Bottom: values.MarginPadding2,
							}.Layout(gtx, pg.searchEditor.Layout)
						})
					}),
					layout.Expanded(func(gtx C) D {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return layout.E.Layout(gtx, func(gtx C) D {
							card := pg.Theme.Card()
							card.Radius = decredmaterial.Radius(8)
							return card.Layout(gtx, func(gtx C) D {
								return layout.UniformInset(values.MarginPadding8).Layout(gtx, func(gtx C) D {
									return pg.layoutSyncSection(gtx)
								})
							})
						})
					}),
					layout.Expanded(func(gtx C) D {
						return pg.orderDropDown.Layout(gtx, 45, true)
					}),
					layout.Expanded(func(gtx C) D {
						return pg.walletDropDown.Layout(gtx, pg.orderDropDown.Width+41, true)
					}),
				)
			})
		}),
	)
}

func (pg *ConsensusPage) lineSeparator(inset layout.Inset) layout.Widget {
	return func(gtx C) D {
		return inset.Layout(gtx, pg.Theme.Separator().Layout)
	}
}

func (pg *ConsensusPage) layoutRedirectVoting(gtx C) D {
	return layout.Flex{Axis: layout.Vertical, Alignment: layout.End}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return pg.viewVotingDashboard.Layout(gtx, func(gtx C) D {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						return layout.Inset{
							Right: values.MarginPadding10,
						}.Layout(gtx, pg.redirectIcon.Layout16dp)
					}),
					layout.Rigid(func(gtx C) D {
						return layout.Inset{
							Top: values.MarginPaddingMinus2,
						}.Layout(gtx, pg.Theme.Label(values.TextSize16, "Voting Dashboard").Layout)
					}),
				)
			})
		}),
		layout.Rigid(func(gtx C) D {
			var text string
			if pg.isSyncing {
				text = "Syncing..."
			} else if pg.syncCompleted {
				text = "Updated"
			}

			lastUpdatedInfo := pg.Theme.Label(values.TextSize10, text)
			lastUpdatedInfo.Color = pg.Theme.Color.GrayText2
			if pg.syncCompleted {
				lastUpdatedInfo.Color = pg.Theme.Color.Success
			}

			return layout.E.Layout(gtx, func(gtx C) D {
				return layout.Inset{Top: values.MarginPadding2}.Layout(gtx, lastUpdatedInfo.Layout)
			})
		}),
	)
}

func (pg *ConsensusPage) layoutContent(gtx C) D {
	if len(pg.consensusItems) == 0 {
		return components.LayoutNoAgendasFound(gtx, pg.Load, pg.isSyncing)
	}
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx C) D {
			list := layout.List{Axis: layout.Vertical}
			return pg.Theme.List(pg.listContainer).Layout(gtx, 1, func(gtx C, i int) D {
				return layout.Inset{Right: values.MarginPadding2}.Layout(gtx, func(gtx C) D {
					return list.Layout(gtx, len(pg.consensusItems), func(gtx C, i int) D {
						return decredmaterial.LinearLayout{
							Orientation: layout.Vertical,
							Width:       decredmaterial.MatchParent,
							Height:      decredmaterial.WrapContent,
							Background:  pg.Theme.Color.Surface,
							Direction:   layout.W,
							Border:      decredmaterial.Border{Radius: decredmaterial.Radius(14)},
							Padding:     layout.UniformInset(values.MarginPadding15),
							Margin:      layout.Inset{Bottom: values.MarginPadding4, Top: values.MarginPadding4}}.
							Layout2(gtx, func(gtx C) D {
								return components.AgendaItemWidget(gtx, pg.Load, pg.consensusItems[i])
							})
					})
				})
			})
		}),
	)
}

func (pg *ConsensusPage) layoutSyncSection(gtx C) D {
	if pg.isSyncing {
		return pg.layoutIsSyncingSection(gtx)
	} else if pg.syncCompleted {
		updatedIcon := decredmaterial.NewIcon(pg.Icons.NavigationCheck)
		updatedIcon.Color = pg.Theme.Color.Success
		return updatedIcon.Layout(gtx, values.MarginPadding20)
	}
	return pg.layoutStartSyncSection(gtx)
}

func (pg *ConsensusPage) layoutIsSyncingSection(gtx C) D {
	th := material.NewTheme(gofont.Collection())
	gtx.Constraints.Max.X = gtx.Px(values.MarginPadding24)
	gtx.Constraints.Min.X = gtx.Constraints.Max.X
	loader := material.Loader(th)
	loader.Color = pg.Theme.Color.Gray1
	return loader.Layout(gtx)
}

func (pg *ConsensusPage) layoutStartSyncSection(gtx C) D {
	// TODO: use decredmaterial clickable
	return material.Clickable(gtx, pg.syncButton, pg.Icons.Restore.Layout24dp)
}
