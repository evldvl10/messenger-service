package router

import (
	"strconv"
	"time"

	"messenger-service/database"
	"messenger-service/model"
	"messenger-service/socketio"
	"messenger-service/utils"

	"github.com/zishang520/socket.io/v2/socket"
)

type InitConnection struct {
	Dialogs    []MessengerDialog     `json:"dialogs"`
	UserStatus []MessengerUserStatus `json:"userStatus"`
}

// Messenger
type MessengerMessage struct {
	Id       uint          `json:"id"`
	Created  time.Time     `json:"created"`
	Dialog   uint          `json:"dialog"`
	From     MessengerUser `json:"from"`
	To       MessengerUser `json:"to"`
	Type     string        `json:"type"`
	Metadata string        `json:"metadata"`
	Data     string        `json:"data"`
	Read     bool          `json:"read"`
}

type MessengerDialog struct {
	Id      uint             `json:"id"`
	Owner   MessengerUser    `json:"owner"`
	User    MessengerUser    `json:"user"`
	Message MessengerMessage `json:"message"`
}

type MessengerUser struct {
	Id       uint   `json:"id"`
	Username string `json:"username"`
}

type MessengerDialogDetails struct {
	Details  MessengerDialog    `json:"details"`
	Messages []MessengerMessage `json:"messages"`
}

type MessengerUserStatus struct {
	Id     uint `json:"id"`
	Status bool `json:"status"`
}

func Socket(server *socket.Server) {
	server.On("connection", func(clients ...interface{}) {
		client := clients[0].(*socket.Socket)

		client.On("init", func(args ...interface{}) {
			rooms := server.Sockets().Adapter().Rooms().Keys()

			// UserStatus
			// Dialogs
			userStatus := []MessengerUserStatus{}
			dialogs := []MessengerDialog{}
			if client.Data() != nil {
				// Get [from] user
				rawDialogs := []model.MessengerDialog{}
				owner, _ := strconv.Atoi(client.Data().(*utils.TokenMetadata).Id)
				database.Postgres.Where(&model.MessengerDialog{OwnerID: owner}).Preload("Owner").Preload("User").Preload("Message").Preload("Message.From").Preload("Message.To").Find(&rawDialogs)

				for _, dialog := range rawDialogs {
					dialogs = append(dialogs, MessengerDialog{
						Id: dialog.ID,
						Owner: MessengerUser{
							Id:       dialog.Owner.ID,
							Username: dialog.Owner.Username,
						},
						User: MessengerUser{
							Id:       dialog.User.ID,
							Username: dialog.User.Username,
						},
						Message: MessengerMessage{
							Id:      dialog.Message.ID,
							Created: dialog.Message.CreatedAt,
							Dialog:  dialog.ID,
							From: MessengerUser{
								Id:       dialog.Message.From.ID,
								Username: dialog.Message.From.Username,
							},
							To: MessengerUser{
								Id:       dialog.Message.To.ID,
								Username: dialog.Message.To.Username,
							},
							Type:     dialog.Message.Type,
							Metadata: dialog.Message.Metadata,
							Data:     dialog.Message.Data,
							Read:     dialog.Message.Read,
						},
					})

					online := false
					for i := range rooms {
						if rooms[i] == socket.Room(strconv.FormatUint(uint64(dialog.User.ID), 10)) {
							online = true
							break
						}
					}

					userStatus = append(userStatus, MessengerUserStatus{
						Id:     dialog.User.ID,
						Status: online,
					})
				}
			}

			// Send response
			client.Emit(
				"init",
				InitConnection{
					Dialogs:    dialogs,
					UserStatus: userStatus,
				},
			)
		})

		client.On("messenger_dialog_create", func(args ...interface{}) {
			from, _ := strconv.Atoi(client.Data().(*utils.TokenMetadata).Id)
			to, _ := strconv.Atoi(args[0].(string))

			if from == to {
				return
			}

			_type := args[1].(string)
			data := args[2].(string)
			_data := ""

			switch _type {
			case "text":
				_data = data
			case "image":
				image := new(model.MessengerImage)
				image.Data = data
				database.Postgres.Create(&image)
				_data = strconv.FormatUint(uint64(image.ID), 10)
			}

			// Get [from] user
			fromUser := new(model.User)
			database.Postgres.First(&fromUser, from)

			// Get [to] user
			toUser := new(model.User)
			database.Postgres.First(&toUser, args[0])

			// Create [from] message
			messageFrom := new(model.MessengerMessage)
			messageFrom.From = *fromUser
			messageFrom.To = *toUser
			messageFrom.Type = _type
			messageFrom.Metadata = ""
			messageFrom.Data = _data
			messageFrom.Read = true
			database.Postgres.Create(&messageFrom)

			// Create [from] dialog
			dialogFrom := new(model.MessengerDialog)
			dialogFrom.Owner = *fromUser
			dialogFrom.User = *toUser
			dialogFrom.Message = *messageFrom
			database.Postgres.Create(&dialogFrom)

			// Update [from] message
			messageFrom.DialogID = dialogFrom.ID
			database.Postgres.Save(&messageFrom)

			// Create [to] message
			messageTo := new(model.MessengerMessage)
			messageTo.From = *fromUser
			messageTo.To = *toUser
			messageTo.Type = _type
			messageTo.Metadata = ""
			messageTo.Data = _data
			messageTo.Read = false
			database.Postgres.Create(&messageTo)

			// Create [to] dialog
			dialogTo := new(model.MessengerDialog)
			dialogTo.Owner = *toUser
			dialogTo.User = *fromUser
			dialogTo.Message = *messageTo
			database.Postgres.Create(&dialogTo)

			// Update [to] message
			messageTo.DialogID = dialogTo.ID
			database.Postgres.Save(&messageTo)

			client.Emit(
				"messenger_dialog_create",
				MessengerDialog{
					Id: dialogFrom.ID,
					Owner: MessengerUser{
						Id:       dialogFrom.Owner.ID,
						Username: dialogFrom.Owner.Username,
					},
					User: MessengerUser{
						Id:       dialogFrom.User.ID,
						Username: dialogFrom.User.Username,
					},
					Message: MessengerMessage{
						Id:      dialogFrom.Message.ID,
						Created: dialogFrom.Message.CreatedAt,
						Dialog:  dialogFrom.ID,
						From: MessengerUser{
							Id:       dialogFrom.Message.From.ID,
							Username: dialogFrom.Message.From.Username,
						},
						To: MessengerUser{
							Id:       dialogFrom.Message.To.ID,
							Username: dialogFrom.Message.To.Username,
						},
						Type:     dialogFrom.Message.Type,
						Metadata: dialogFrom.Message.Metadata,
						Data:     dialogFrom.Message.Data,
						Read:     dialogFrom.Message.Read,
					},
				},
			)

			socketio.Emit(
				args[0].(string),
				"messenger_dialog_create",
				MessengerDialog{
					Id: dialogTo.ID,
					Owner: MessengerUser{
						Id:       dialogTo.Owner.ID,
						Username: dialogTo.Owner.Username,
					},
					User: MessengerUser{
						Id:       dialogTo.User.ID,
						Username: dialogTo.User.Username,
					},
					Message: MessengerMessage{
						Id:      dialogTo.Message.ID,
						Created: dialogTo.Message.CreatedAt,
						Dialog:  dialogTo.ID,
						From: MessengerUser{
							Id:       dialogTo.Message.From.ID,
							Username: dialogTo.Message.From.Username,
						},
						To: MessengerUser{
							Id:       dialogTo.Message.To.ID,
							Username: dialogTo.Message.To.Username,
						},
						Type:     dialogTo.Message.Type,
						Metadata: dialogTo.Message.Metadata,
						Data:     dialogTo.Message.Data,
						Read:     dialogTo.Message.Read,
					},
				},
			)
		})

		client.On("messenger_dialog_messages", func(args ...interface{}) {
			dialog, _ := strconv.ParseUint(args[0].(string), 10, 64)

			messages := []MessengerMessage{}
			rawMessages := []model.MessengerMessage{}
			database.Postgres.Order("ID asc").Where(&model.MessengerMessage{DialogID: uint(dialog)}).Preload("From").Preload("To").Find(&rawMessages)

			for _, message := range rawMessages {
				messages = append(messages, MessengerMessage{
					Id:      message.ID,
					Created: message.CreatedAt,
					Dialog:  message.DialogID,
					From: MessengerUser{
						Id:       message.From.ID,
						Username: message.From.Username,
					},
					To: MessengerUser{
						Id:       message.To.ID,
						Username: message.To.Username,
					},
					Type:     message.Type,
					Metadata: message.Metadata,
					Data:     message.Data,
					Read:     message.Read,
				})
			}

			database.Postgres.Model(&model.MessengerMessage{}).Where(&model.MessengerMessage{DialogID: uint(dialog)}).Update("read", true)

			// Get [from] dialog
			fromDialog := model.MessengerDialog{}
			database.Postgres.Preload("Owner").Preload("User").Preload("Message").Find(&fromDialog, dialog)

			client.Emit(
				"messenger_dialog_messages",
				MessengerDialogDetails{
					Details: MessengerDialog{
						Id: fromDialog.ID,
						Owner: MessengerUser{
							Id:       fromDialog.Owner.ID,
							Username: fromDialog.Owner.Username,
						},
						User: MessengerUser{
							Id:       fromDialog.User.ID,
							Username: fromDialog.User.Username,
						},
						Message: MessengerMessage{
							Id:      fromDialog.Message.ID,
							Created: fromDialog.Message.CreatedAt,
							Dialog:  fromDialog.ID,
							From: MessengerUser{
								Id:       fromDialog.Message.From.ID,
								Username: fromDialog.Message.From.Username,
							},
							To: MessengerUser{
								Id:       fromDialog.Message.To.ID,
								Username: fromDialog.Message.To.Username,
							},
							Type:     fromDialog.Message.Type,
							Metadata: fromDialog.Message.Metadata,
							Data:     fromDialog.Message.Data,
							Read:     fromDialog.Message.Read,
						},
					},
					Messages: messages,
				},
			)
		})

		client.On("messenger_dialog_list", func(args ...interface{}) {
			dialogs := []MessengerDialog{}
			rawDialogs := []model.MessengerDialog{}
			owner, _ := strconv.Atoi(client.Data().(*utils.TokenMetadata).Id)
			database.Postgres.Where(&model.MessengerDialog{OwnerID: owner}).Preload("Owner").Preload("User").Preload("Message").Preload("Message.From").Preload("Message.To").Find(&rawDialogs)

			for _, dialog := range rawDialogs {
				dialogs = append(dialogs, MessengerDialog{
					Id: dialog.ID,
					Owner: MessengerUser{
						Id:       dialog.Owner.ID,
						Username: dialog.Owner.Username,
					},
					User: MessengerUser{
						Id:       dialog.User.ID,
						Username: dialog.User.Username,
					},
					Message: MessengerMessage{
						Id:      dialog.Message.ID,
						Created: dialog.Message.CreatedAt,
						Dialog:  dialog.ID,
						From: MessengerUser{
							Id:       dialog.Message.From.ID,
							Username: dialog.Message.From.Username,
						},
						To: MessengerUser{
							Id:       dialog.Message.To.ID,
							Username: dialog.Message.To.Username,
						},
						Type:     dialog.Message.Type,
						Metadata: dialog.Message.Metadata,
						Data:     dialog.Message.Data,
						Read:     dialog.Message.Read,
					},
				})
			}

			client.Emit(
				"messenger_dialog_list",
				dialogs,
			)
		})

		client.On("messenger_send_message", func(args ...interface{}) {
			dialog, _ := strconv.ParseUint(args[0].(string), 10, 64)
			_type := args[1].(string)
			data := args[2].(string)
			from, _ := strconv.Atoi(client.Data().(*utils.TokenMetadata).Id)

			_data := ""

			switch _type {
			case "text":
				_data = data
			case "image":
				image := new(model.MessengerImage)
				image.Data = data
				database.Postgres.Create(&image)
				_data = strconv.FormatUint(uint64(image.ID), 10)
			}

			// Get [from] dialog
			fromDialog := model.MessengerDialog{}
			database.Postgres.Where(&model.MessengerDialog{OwnerID: from}).Preload("Owner").Preload("User").Preload("Message").Find(&fromDialog, dialog)

			// Get [to] dialog
			toDialog := model.MessengerDialog{}
			database.Postgres.Where(&model.MessengerDialog{OwnerID: fromDialog.UserID, UserID: from}).Preload("Owner").Preload("User").Preload("Message").Find(&toDialog)

			// Get [from] user
			fromUser := new(model.User)
			database.Postgres.First(&fromUser, from)

			// Get [to] user
			toUser := new(model.User)
			database.Postgres.First(&toUser, fromDialog.UserID)

			// Create [from] message
			messageFrom := new(model.MessengerMessage)
			messageFrom.DialogID = fromDialog.ID
			messageFrom.From = *fromUser
			messageFrom.To = *toUser
			messageFrom.Type = _type
			messageFrom.Metadata = ""
			messageFrom.Data = _data
			messageFrom.Read = true
			database.Postgres.Create(&messageFrom)

			// Update [from] dialog
			fromDialog.Message = *messageFrom
			database.Postgres.Save(&fromDialog)

			// Create [to] message
			messageTo := new(model.MessengerMessage)
			messageTo.DialogID = toDialog.ID
			messageTo.From = *fromUser
			messageTo.To = *toUser
			messageTo.Type = _type
			messageTo.Metadata = ""
			messageTo.Data = _data
			messageTo.Read = false
			database.Postgres.Create(&messageTo)

			// Update [to] dialog
			toDialog.Message = *messageTo
			database.Postgres.Save(&toDialog)

			client.Emit(
				"messenger_send_message",
				MessengerMessage{
					Id:      messageFrom.ID,
					Created: messageFrom.CreatedAt,
					Dialog:  messageFrom.DialogID,
					From: MessengerUser{
						Id:       messageFrom.From.ID,
						Username: messageFrom.From.Username,
					},
					To: MessengerUser{
						Id:       messageFrom.To.ID,
						Username: messageFrom.To.Username,
					},
					Type:     messageFrom.Type,
					Metadata: messageFrom.Metadata,
					Data:     messageFrom.Data,
					Read:     messageFrom.Read,
				},
			)

			socketio.Emit(
				strconv.Itoa(fromDialog.UserID),
				"messenger_send_message",
				MessengerMessage{
					Id:      messageTo.ID,
					Created: messageTo.CreatedAt,
					Dialog:  messageTo.DialogID,
					From: MessengerUser{
						Id:       messageTo.From.ID,
						Username: messageTo.From.Username,
					},
					To: MessengerUser{
						Id:       messageTo.To.ID,
						Username: messageTo.To.Username,
					},
					Type:     messageTo.Type,
					Metadata: messageTo.Metadata,
					Data:     messageTo.Data,
					Read:     messageTo.Read,
				},
			)
		})

		client.On("messenger_read_dialog", func(args ...interface{}) {
			dialog, _ := strconv.ParseUint(args[0].(string), 10, 64)
			database.Postgres.Model(&model.MessengerMessage{}).Where(&model.MessengerMessage{DialogID: uint(dialog)}).Update("read", true)
		})

		client.On("messenger_user_status", func(args ...interface{}) {
			rooms := server.Sockets().Adapter().Rooms().Keys()

			userStatus := []MessengerUserStatus{}
			if client.Data() != nil {
				rawDialogs := []model.MessengerDialog{}
				owner, _ := strconv.Atoi(client.Data().(*utils.TokenMetadata).Id)
				database.Postgres.Where(&model.MessengerDialog{OwnerID: owner}).Preload("Owner").Preload("User").Preload("Message").Find(&rawDialogs)

				for _, dialog := range rawDialogs {
					online := false
					for i := range rooms {
						if rooms[i] == socket.Room(strconv.FormatUint(uint64(dialog.User.ID), 10)) {
							online = true
							break
						}
					}

					userStatus = append(userStatus, MessengerUserStatus{
						Id:     dialog.User.ID,
						Status: online,
					})
				}
			}

			// Send response
			client.Emit(
				"messenger_user_status",
				userStatus,
			)
		})
	})
}
