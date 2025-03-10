package page

import (
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/planetdecred/dcrlibwallet"
	"github.com/planetdecred/godcr/ui/decredmaterial"
	"github.com/planetdecred/godcr/ui/load"
	"github.com/planetdecred/godcr/ui/modal"
	"github.com/planetdecred/godcr/ui/page/components"
	"github.com/planetdecred/godcr/ui/preference"
	"github.com/planetdecred/godcr/ui/values"
	"github.com/planetdecred/godcr/wallet"
)

const SettingsPageID = "Settings"

type row struct {
	title     string
	clickable *decredmaterial.Clickable
	icon      *decredmaterial.Icon
	label     decredmaterial.Label
}

type SettingsPage struct {
	*load.Load

	pageContainer *widget.List
	wal           *wallet.Wallet

	updateConnectToPeer *decredmaterial.Clickable
	updateUserAgent     *decredmaterial.Clickable
	changeStartupPass   *decredmaterial.Clickable
	language            *decredmaterial.Clickable
	currency            *decredmaterial.Clickable

	chevronRightIcon *decredmaterial.Icon
	backButton       decredmaterial.IconButton
	infoButton       decredmaterial.IconButton

	isDarkModeOn            *decredmaterial.Switch
	spendUnconfirmed        *decredmaterial.Switch
	startupPassword         *decredmaterial.Switch
	beepNewBlocks           *decredmaterial.Switch
	connectToPeer           *decredmaterial.Switch
	userAgent               *decredmaterial.Switch
	governance              *decredmaterial.Switch
	proposalNotification    *decredmaterial.Switch
	transactionNotification *decredmaterial.Switch

	peerLabel, agentLabel decredmaterial.Label

	isStartupPassword bool
	peerAddr          string
	agentValue        string
	errorReceiver     chan error
}

func NewSettingsPage(l *load.Load) *SettingsPage {
	chevronRightIcon := l.Icons.ChevronRight

	pg := &SettingsPage{
		Load: l,
		pageContainer: &widget.List{
			List: layout.List{Axis: layout.Vertical},
		},
		wal: l.WL.Wallet,

		isDarkModeOn:            l.Theme.Switch(),
		spendUnconfirmed:        l.Theme.Switch(),
		startupPassword:         l.Theme.Switch(),
		beepNewBlocks:           l.Theme.Switch(),
		connectToPeer:           l.Theme.Switch(),
		userAgent:               l.Theme.Switch(),
		governance:              l.Theme.Switch(),
		proposalNotification:    l.Theme.Switch(),
		transactionNotification: l.Theme.Switch(),

		chevronRightIcon: decredmaterial.NewIcon(chevronRightIcon),

		errorReceiver: make(chan error),

		updateConnectToPeer: l.Theme.NewClickable(false),
		updateUserAgent:     l.Theme.NewClickable(false),
		changeStartupPass:   l.Theme.NewClickable(false),
		language:            l.Theme.NewClickable(false),
		currency:            l.Theme.NewClickable(false),
	}

	pg.backButton, pg.infoButton = components.SubpageHeaderButtons(l)

	return pg
}

// ID is a unique string that identifies the page and may be used
// to differentiate this page from other pages.
// Part of the load.Page interface.
func (pg *SettingsPage) ID() string {
	return SettingsPageID
}

// OnNavigatedTo is called when the page is about to be displayed and
// may be used to initialize page features that are only relevant when
// the page is displayed.
// Part of the load.Page interface.
func (pg *SettingsPage) OnNavigatedTo() {

}

// Layout draws the page UI components into the provided layout context
// to be eventually drawn on screen.
// Part of the load.Page interface.
func (pg *SettingsPage) Layout(gtx layout.Context) layout.Dimensions {
	pg.updateSettingOptions()

	body := func(gtx C) D {
		sp := components.SubPage{
			Load:       pg.Load,
			Title:      values.String(values.StrSettings),
			BackButton: pg.backButton,
			Back: func() {
				pg.PopFragment()
			},
			Body: func(gtx layout.Context) layout.Dimensions {
				pageContent := []func(gtx C) D{
					pg.general(),
					pg.security(),
					pg.notification(),
					pg.connection(),
				}

				return pg.Theme.List(pg.pageContainer).Layout(gtx, len(pageContent), func(gtx C, i int) D {
					return layout.Inset{Right: values.MarginPadding2}.Layout(gtx, pageContent[i])
				})
			},
		}
		return sp.Layout(gtx)
	}

	return components.UniformPadding(gtx, body)
}

func (pg *SettingsPage) general() layout.Widget {
	return func(gtx C) D {
		return pg.mainSection(gtx, values.String(values.StrGeneral), func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return pg.subSectionSwitch(gtx, "Dark mode", pg.isDarkModeOn)
				}),
				layout.Rigid(func(gtx C) D {
					return pg.subSectionSwitch(gtx, values.String(values.StrUnconfirmedFunds), pg.spendUnconfirmed)
				}),
				layout.Rigid(func(gtx C) D {
					return pg.subSectionSwitch(gtx, "Governance", pg.governance)
				}),
				layout.Rigid(pg.lineSeparator()),
				layout.Rigid(func(gtx C) D {
					currencyConversionRow := row{
						title:     values.String(values.StrCurrencyConversion),
						clickable: pg.currency,
						icon:      pg.chevronRightIcon,
						label:     pg.Theme.Body2(pg.wal.ReadStringConfigValueForKey(dcrlibwallet.CurrencyConversionConfigKey)),
					}
					return pg.clickableRow(gtx, currencyConversionRow)
				}),
				layout.Rigid(pg.lineSeparator()),
				layout.Rigid(func(gtx C) D {
					languageRow := row{
						title:     values.String(values.StrLanguage),
						clickable: pg.language,
						icon:      pg.chevronRightIcon,
						label:     pg.Theme.Body2(pg.wal.ReadStringConfigValueForKey(load.LanguagePreferenceKey)),
					}
					return pg.clickableRow(gtx, languageRow)
				}),
			)
		})
	}
}

func (pg *SettingsPage) notification() layout.Widget {
	return func(gtx C) D {
		return pg.mainSection(gtx, values.String(values.StrNotifications), func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return pg.subSectionSwitch(gtx, values.String(values.StrBeepForNewBlocks), pg.beepNewBlocks)
				}),
				layout.Rigid(func(gtx C) D {
					return pg.subSectionSwitch(gtx, "Transaction notification", pg.transactionNotification)
				}),
				layout.Rigid(func(gtx C) D {
					return pg.subSectionSwitch(gtx, "Proposal notification", pg.proposalNotification)
				}),
			)
		})
	}
}

func (pg *SettingsPage) security() layout.Widget {
	return func(gtx C) D {
		return pg.mainSection(gtx, values.String(values.StrSecurity), func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return pg.subSectionSwitch(gtx, values.String(values.StrStartupPassword), pg.startupPassword)
				}),
				layout.Rigid(func(gtx C) D {
					return pg.conditionalDisplay(gtx, pg.isStartupPassword, func(gtx C) D {
						changeStartupPassRow := row{
							title:     values.String(values.StrChangeStartupPassword),
							clickable: pg.changeStartupPass,
							icon:      pg.chevronRightIcon,
							label:     pg.Theme.Body1(""),
						}
						return pg.clickableRow(gtx, changeStartupPassRow)
					})
				}),
			)
		})
	}
}

func (pg *SettingsPage) connection() layout.Widget {
	return func(gtx C) D {
		return pg.mainSection(gtx, values.String(values.StrConnection), func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return pg.subSectionSwitch(gtx, values.String(values.StrConnectToSpecificPeer), pg.connectToPeer)
				}),
				layout.Rigid(func(gtx C) D {
					peerLabel := pg.Theme.Body1(pg.peerAddr)
					peerLabel.Color = pg.Theme.Color.GrayText2
					peerAddrRow := row{
						title:     values.String(values.StrChangeSpecificPeer),
						clickable: pg.updateConnectToPeer,
						icon:      pg.chevronRightIcon,
						label:     peerLabel,
					}
					return pg.conditionalDisplay(gtx, pg.peerAddr != "", func(gtx C) D {
						return pg.clickableRow(gtx, peerAddrRow)
					})
				}),
				layout.Rigid(pg.lineSeparator()),
				layout.Rigid(pg.agent()),
			)
		})
	}
}

func (pg *SettingsPage) agent() layout.Widget {
	return func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Flex{}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						m10 := values.MarginPadding10
						return layout.Inset{Top: m10, Bottom: m10}.Layout(gtx, func(gtx C) D {
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(pg.subSectionLabel(values.String(values.StrCustomUserAgent))),
								layout.Rigid(func(gtx C) D {
									txt := pg.Theme.Body2("For HTTP request")
									txt.Color = pg.Theme.Color.GrayText2
									return layout.Inset{Top: values.MarginPadding5}.Layout(gtx, func(gtx C) D {
										return txt.Layout(gtx)
									})
								}),
							)
						})
					}),
					layout.Flexed(1, func(gtx C) D {
						return layout.Inset{Top: values.MarginPadding7}.Layout(gtx, func(gtx C) D {
							return layout.E.Layout(gtx, pg.userAgent.Layout)
						})
					}),
				)
			}),
			layout.Rigid(func(gtx C) D {
				agentLabel := pg.Theme.Body1(pg.agentValue)
				agentLabel.Color = pg.Theme.Color.GrayText2
				return pg.conditionalDisplay(gtx, pg.agentValue != "", func(gtx C) D {
					userAgentRow := row{
						title:     values.String(values.StrUserAgentDialogTitle),
						clickable: pg.updateUserAgent,
						icon:      pg.chevronRightIcon,
						label:     agentLabel,
					}
					return pg.clickableRow(gtx, userAgentRow)
				})
			}),
		)
	}
}

func (pg *SettingsPage) mainSection(gtx layout.Context, title string, body layout.Widget) layout.Dimensions {
	return layout.Inset{Bottom: values.MarginPadding10}.Layout(gtx, func(gtx C) D {
		return pg.Theme.Card().Layout(gtx, func(gtx C) D {
			return layout.UniformInset(values.MarginPadding15).Layout(gtx, func(gtx C) D {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								txt := pg.Theme.Body2(title)
								txt.Color = pg.Theme.Color.GrayText2
								return layout.Inset{Bottom: values.MarginPadding10}.Layout(gtx, txt.Layout)
							}),
							layout.Flexed(1, func(gtx C) D {
								if title == values.String(values.StrSecurity) {
									pg.infoButton.Inset = layout.UniformInset(values.MarginPadding0)
									pg.infoButton.Size = values.MarginPadding20
									return layout.E.Layout(gtx, pg.infoButton.Layout)
								}
								return D{}
							}),
						)
					}),
					layout.Rigid(body),
				)
			})
		})
	})
}

func (pg *SettingsPage) subSection(gtx layout.Context, title string, body layout.Widget) layout.Dimensions {
	return layout.Inset{Top: values.MarginPadding5, Bottom: values.MarginPadding15}.Layout(gtx, func(gtx C) D {
		return layout.Flex{}.Layout(gtx,
			layout.Rigid(pg.subSectionLabel(title)),
			layout.Flexed(1, func(gtx C) D {
				return layout.E.Layout(gtx, body)
			}),
		)
	})
}

func (pg *SettingsPage) subSectionSwitch(gtx layout.Context, title string, option *decredmaterial.Switch) layout.Dimensions {
	return pg.subSection(gtx, title, option.Layout)
}

func (pg *SettingsPage) clickableRow(gtx layout.Context, row row) layout.Dimensions {
	return row.clickable.Layout(gtx, func(gtx C) D {
		return layout.Inset{Top: values.MarginPadding5, Bottom: values.MarginPaddingMinus5}.Layout(gtx, func(gtx C) D {
			return pg.subSection(gtx, row.title, func(gtx C) D {
				return layout.Flex{}.Layout(gtx,
					layout.Rigid(row.label.Layout),
					layout.Rigid(func(gtx C) D {
						ic := row.icon
						ic.Color = pg.Theme.Color.Gray3
						return ic.Layout(gtx, values.MarginPadding22)
					}),
				)
			})
		})
	})
}

func (pg *SettingsPage) conditionalDisplay(gtx layout.Context, display bool, body layout.Widget) layout.Dimensions {
	if display {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(pg.lineSeparator()),
			layout.Rigid(body),
		)
	}
	return layout.Dimensions{}
}

func (pg *SettingsPage) subSectionLabel(title string) layout.Widget {
	return func(gtx C) D {
		return pg.Theme.Body1(title).Layout(gtx)
	}
}

func (pg *SettingsPage) lineSeparator() layout.Widget {
	m := values.MarginPadding1
	return func(gtx C) D {
		return layout.Inset{Top: m, Bottom: m}.Layout(gtx, pg.Theme.Separator().Layout)
	}
}

func (pg *SettingsPage) showWarningModalDialog(title, msg, key string) {
	info := modal.NewInfoModal(pg.Load).
		Title(title).
		Body(msg).
		NegativeButton(values.String(values.StrCancel), func() {}).
		PositiveButtonStyle(pg.Theme.Color.Surface, pg.Theme.Color.Danger).
		PositiveButton("Remove", func() {
			pg.WL.MultiWallet.DeleteUserConfigValueForKey(key)
		})
	pg.ShowModal(info)
}

// HandleUserInteractions is called just before Layout() to determine
// if any user interaction recently occurred on the page and may be
// used to update the page's UI components shortly before they are
// displayed.
// Part of the load.Page interface.
func (pg *SettingsPage) HandleUserInteractions() {

	for pg.language.Clicked() {
		preference.NewListPreference(pg.WL.Wallet, pg.Load,
			load.LanguagePreferenceKey, values.DefaultLangauge, values.ArrLanguages).
			Title(values.StrLanguage).
			UpdateValues(func() {
				values.SetUserLanguage(pg.wal.ReadStringConfigValueForKey(load.LanguagePreferenceKey))
			}).Show()
		break
	}

	for pg.currency.Clicked() {
		preference.NewListPreference(pg.WL.Wallet, pg.Load,
			dcrlibwallet.CurrencyConversionConfigKey, values.DefaultExchangeValue,
			values.ArrExchangeCurrencies).
			Title(values.StrCurrencyConversion).
			UpdateValues(func() {}).
			Show()
		break
	}

	if pg.isDarkModeOn.Changed() {
		pg.wal.SaveConfigValueForKey(load.DarkModeConfigKey, pg.isDarkModeOn.IsChecked())
		pg.RefreshTheme()
	}

	if pg.spendUnconfirmed.Changed() {
		pg.wal.SaveConfigValueForKey(dcrlibwallet.SpendUnconfirmedConfigKey, pg.spendUnconfirmed.IsChecked())
	}

	if pg.governance.Changed() {
		if pg.governance.IsChecked() {
			go pg.WL.MultiWallet.Politeia.Sync()
			pg.WL.Wallet.SaveConfigValueForKey(load.FetchProposalConfigKey, pg.governance.IsChecked())
			pg.Toast.Notify("Proposals fetching enabled. Check Governance page")
		} else {
			info := modal.NewInfoModal(pg.Load).
				Title("Governance").
				Body("Are you sure you want to disable governance? This will clear all available proposals").
				NegativeButton(values.String(values.StrCancel), func() {}).
				PositiveButtonStyle(pg.Theme.Color.Surface, pg.Theme.Color.Danger).
				PositiveButton("Disable", func() {
					if pg.WL.MultiWallet.Politeia.IsSyncing() {
						go pg.WL.MultiWallet.Politeia.StopSync()
					}
					pg.wal.SaveConfigValueForKey(load.FetchProposalConfigKey, !pg.governance.IsChecked())
					pg.WL.MultiWallet.Politeia.ClearSavedProposals()
					pg.Toast.Notify("Proposals fetching Disabled.")
				})
			pg.ShowModal(info)
		}
	}

	if pg.beepNewBlocks.Changed() {
		pg.wal.SaveConfigValueForKey(dcrlibwallet.BeepNewBlocksConfigKey, pg.beepNewBlocks.IsChecked())
	}

	if pg.proposalNotification.Changed() {
		pg.wal.SaveConfigValueForKey(load.ProposalNotificationConfigKey, pg.proposalNotification.IsChecked())
		if pg.proposalNotification.IsChecked() {
			pg.Toast.Notify("Proposal notification enabled")
		} else {
			pg.Toast.Notify("Proposal notification disabled")
		}
	}

	if pg.transactionNotification.Changed() {
		pg.wal.SaveConfigValueForKey(load.TransactionNotificationConfigKey, pg.transactionNotification.IsChecked())
		if pg.transactionNotification.IsChecked() {
			pg.Toast.Notify("Transaction notification enabled")
		} else {
			pg.Toast.Notify("Transaction notification disabled")
		}
	}

	if pg.infoButton.Button.Clicked() {
		info := modal.NewInfoModal(pg.Load).
			Title("Set up startup password").
			Body("Startup password helps protect your wallet from unauthorized access.").
			PositiveButton("Got it", func() {})
		pg.ShowModal(info)
	}

	for pg.changeStartupPass.Clicked() {
		modal.NewPasswordModal(pg.Load).
			Title("Confirm current startup password").
			Hint("Current startup password").
			NegativeButton(values.String(values.StrCancel), func() {}).
			PositiveButton(values.String(values.StrConfirm), func(password string, pm *modal.PasswordModal) bool {
				go func() {
					var error string
					err := pg.wal.GetMultiWallet().VerifyStartupPassphrase([]byte(password))
					if err != nil {
						if err.Error() == dcrlibwallet.ErrInvalidPassphrase {
							error = "Invalid password"
						} else {
							error = err.Error()
						}
						pm.SetError(error)
						pm.SetLoading(false)
						return
					}
					pm.Dismiss()

					// change password
					modal.NewCreatePasswordModal(pg.Load).
						Title(values.String(values.StrCreateStartupPassword)).
						EnableName(false).
						PasswordHint("New startup password").
						ConfirmPasswordHint("Confirm new startup password").
						PasswordCreated(func(walletName, newPassword string, m *modal.CreatePasswordModal) bool {
							go func() {
								err := pg.wal.GetMultiWallet().ChangeStartupPassphrase([]byte(password), []byte(newPassword), dcrlibwallet.PassphraseTypePass)
								if err != nil {
									m.SetError(err.Error())
									m.SetLoading(false)
									return
								}
								pg.Toast.Notify("Startup password changed")
								m.Dismiss()
							}()
							return false
						}).Show()

				}()

				return false
			}).Show()
		break
	}

	if pg.startupPassword.Changed() {
		if pg.startupPassword.IsChecked() {
			modal.NewCreatePasswordModal(pg.Load).
				Title(values.String(values.StrCreateStartupPassword)).
				EnableName(false).
				PasswordHint("Startup password").
				ConfirmPasswordHint("Confirm startup password").
				PasswordCreated(func(walletName, password string, m *modal.CreatePasswordModal) bool {
					go func() {
						err := pg.wal.GetMultiWallet().SetStartupPassphrase([]byte(password), dcrlibwallet.PassphraseTypePass)
						if err != nil {
							m.SetError(err.Error())
							m.SetLoading(false)
							return
						}
						pg.Toast.Notify("Startup password enabled")
						m.Dismiss()
					}()
					return false
				}).Show()
		} else {
			modal.NewPasswordModal(pg.Load).
				Title(values.String(values.StrConfirmRemoveStartupPass)).
				Hint("Startup password").
				NegativeButton(values.String(values.StrCancel), func() {}).
				PositiveButton(values.String(values.StrConfirm), func(password string, pm *modal.PasswordModal) bool {
					go func() {
						var error string
						err := pg.wal.GetMultiWallet().RemoveStartupPassphrase([]byte(password))
						if err != nil {
							if err.Error() == dcrlibwallet.ErrInvalidPassphrase {
								error = "Invalid password"
							} else {
								error = err.Error()
							}
							pm.SetError(error)
							pm.SetLoading(false)
							return
						}
						pg.Toast.Notify("Startup password disabled")
						pm.Dismiss()
					}()

					return false
				}).Show()
		}
	}

	specificPeerKey := dcrlibwallet.SpvPersistentPeerAddressesConfigKey
	if pg.connectToPeer.Changed() {
		if pg.connectToPeer.IsChecked() {
			pg.showSPVPeerDialog()
			return
		}

		title := "Remove specific peer"
		msg := "Are you sure you want to proceed with removing the specific peer?"
		pg.showWarningModalDialog(title, msg, specificPeerKey)
	}

	for pg.updateConnectToPeer.Clicked() {
		pg.showSPVPeerDialog()
		break
	}

	userAgentKey := dcrlibwallet.UserAgentConfigKey
	for pg.updateUserAgent.Clicked() {
		pg.showUserAgentDialog()
		break
	}

	if pg.userAgent.Changed() {
		if pg.userAgent.IsChecked() {
			pg.showUserAgentDialog()
			return
		}

		title := "Remove user agent"
		msg := "Are you sure you want to proceed with removing the user agent?"
		pg.showWarningModalDialog(title, msg, userAgentKey)
	}

	select {
	case err := <-pg.errorReceiver:
		if err.Error() == dcrlibwallet.ErrInvalidPassphrase {
			e := "Password is incorrect"
			pg.Toast.NotifyError(e)
			return
		}
		pg.Toast.NotifyError(err.Error())
	default:
	}
}

func (pg *SettingsPage) showSPVPeerDialog() {
	textModal := modal.NewTextInputModal(pg.Load).
		Hint("IP address").
		PositiveButtonStyle(pg.Load.Theme.Color.Primary, pg.Load.Theme.Color.InvText).
		PositiveButton(values.String(values.StrConfirm), func(ipAddress string, tim *modal.TextInputModal) bool {
			if ipAddress != "" {
				pg.wal.SaveConfigValueForKey(dcrlibwallet.SpvPersistentPeerAddressesConfigKey, ipAddress)
			}
			return true
		})

	textModal.Title(values.String(values.StrConnectToSpecificPeer)).
		NegativeButton(values.String(values.StrCancel), func() {})
	textModal.Show()
}

func (pg *SettingsPage) showUserAgentDialog() {
	textModal := modal.NewTextInputModal(pg.Load).
		Hint("User agent").
		PositiveButtonStyle(pg.Load.Theme.Color.Primary, pg.Load.Theme.Color.InvText).
		PositiveButton(values.String(values.StrConfirm), func(userAgent string, tim *modal.TextInputModal) bool {
			if userAgent != "" {
				pg.wal.SaveConfigValueForKey(dcrlibwallet.UserAgentConfigKey, userAgent)
			}
			return true
		})

	textModal.Title(values.String(values.StrChangeUserAgent)).
		NegativeButton(values.String(values.StrCancel), func() {})
	textModal.Show()
}

func (pg *SettingsPage) updateSettingOptions() {
	isPassword := pg.WL.MultiWallet.IsStartupSecuritySet()
	pg.startupPassword.SetChecked(false)
	pg.isStartupPassword = false
	if isPassword {
		pg.startupPassword.SetChecked(isPassword)
		pg.isStartupPassword = true
	}

	isDarkModeOn := pg.wal.ReadBoolConfigValueForKey(load.DarkModeConfigKey)
	pg.isDarkModeOn.SetChecked(false)
	if isDarkModeOn {
		pg.isDarkModeOn.SetChecked(isDarkModeOn)
	}

	isSpendUnconfirmed := pg.wal.ReadBoolConfigValueForKey(dcrlibwallet.SpendUnconfirmedConfigKey)
	pg.spendUnconfirmed.SetChecked(false)
	if isSpendUnconfirmed {
		pg.spendUnconfirmed.SetChecked(isSpendUnconfirmed)
	}

	beep := pg.wal.ReadBoolConfigValueForKey(dcrlibwallet.BeepNewBlocksConfigKey)
	pg.beepNewBlocks.SetChecked(false)
	if beep {
		pg.beepNewBlocks.SetChecked(beep)
	}

	pg.peerAddr = pg.wal.ReadStringConfigValueForKey(dcrlibwallet.SpvPersistentPeerAddressesConfigKey)
	pg.connectToPeer.SetChecked(false)
	if pg.peerAddr != "" {
		pg.peerLabel.Text = pg.peerAddr
		pg.connectToPeer.SetChecked(true)
	}

	pg.agentValue = pg.wal.ReadStringConfigValueForKey(dcrlibwallet.UserAgentConfigKey)
	pg.userAgent.SetChecked(false)
	if pg.agentValue != "" {
		pg.agentLabel.Text = pg.agentValue
		pg.userAgent.SetChecked(true)
	}

	governanceSet := pg.wal.ReadBoolConfigValueForKey(load.FetchProposalConfigKey)
	pg.governance.SetChecked(false)
	if governanceSet {
		pg.governance.SetChecked(governanceSet)
	}

	proposalNotification := pg.wal.ReadBoolConfigValueForKey(load.ProposalNotificationConfigKey)
	pg.proposalNotification.SetChecked(false)
	if proposalNotification {
		pg.proposalNotification.SetChecked(proposalNotification)
	}

	transactionNotification := pg.wal.ReadBoolConfigValueForKey(load.TransactionNotificationConfigKey)
	pg.transactionNotification.SetChecked(false)
	if transactionNotification {
		pg.transactionNotification.SetChecked(transactionNotification)
	}
}

// OnNavigatedFrom is called when the page is about to be removed from
// the displayed window. This method should ideally be used to disable
// features that are irrelevant when the page is NOT displayed.
// NOTE: The page may be re-displayed on the app's window, in which case
// OnNavigatedTo() will be called again. This method should not destroy UI
// components unless they'll be recreated in the OnNavigatedTo() method.
// Part of the load.Page interface.
func (pg *SettingsPage) OnNavigatedFrom() {}
