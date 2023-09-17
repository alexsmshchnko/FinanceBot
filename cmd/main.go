package main

import (
	"financebot/internal"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var gBot *tgbotapi.BotAPI
var gToken string
var gChatId int64
var gUsersInChat Users
var gUsefullActivities = Activities{
	// Self-Development
	{"yoga", "Yoga (15 minutes)", 1},
	{"meditation", "Meditation (15 minutes)", 1},
	{"language", "Learning a foreign language (15 minutes)", 1},
	{"swimming", "Swimming (15 minutes)", 1},
	{"walk", "Walk (15 minutes)", 1},
	{"chores", "Chores", 1},

	// Work
	{"work_learning", "Studying work materials (15 minutes)", 1},
	{"portfolio_work", "Working on a portfolio project (15 minutes)", 1},
	{"resume_edit", "Resume editing (15 minutes)", 1},

	// Creativity
	{"creative", "Creative creation (15 minutes)", 1},
	{"reading", "Reading fiction literature (15 minutes)", 1},
}
var gRewards = Activities{
	// Entertainment
	{"watch_series", "Watching a series (1 episode)", 10},
	{"watch_movie", "Watching a movie (1 item)", 30},
	{"social_nets", "Browsing social networks (30 minutes)", 10},

	// Food
	{"eat_sweets", "300 kcal of sweets", 60},
}

type User struct {
	id    int64
	name  string
	coins uint16
}

type Activity struct {
	code, name string
	coins      uint16
}

type Activities []*Activity

type Users []*User

func init() {
	// Uncomment and update token value to set environment variable for Telegram Bot Token given by BotFather.
	// Delete this line after setting the env var. Keep the token out of the public domain!
	//_ = os.Setenv(TokenNameOnOS, "INSERT_YOUR_TOKEN")

	if gToken = os.Getenv(internal.TokenNameOnOS); gToken == "" {
		panic(fmt.Errorf("failed to load env variable %s", internal.TokenNameOnOS))
	}

	var err error
	if gBot, err = tgbotapi.NewBotAPI(gToken); err != nil {
		log.Panic(err)
	}

	gBot.Debug = true
}

func isCallbackQueryNil(update *tgbotapi.Update) bool {
	return update.CallbackQuery != nil && update.CallbackQuery.Data != ""
}

func showMenu() {
	msg := tgbotapi.NewMessage(gChatId, "Choose option")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		getKeyboardRow(internal.BUTTON_TEXT_BALANCE, internal.BUTTON_CODE_BALANCE),
		getKeyboardRow(internal.BUTTON_TEXT_USEFUL_ACTIVITIES, internal.BUTTON_CODE_USEFUL_ACTIVITIES),
		getKeyboardRow(internal.BUTTON_TEXT_REWARDS, internal.BUTTON_CODE_REWARDS),
	)

	gBot.Send(msg)
}

func showBalance(user *User) {
	msg := fmt.Sprintf("%s, wallet is empty now %s\n track activity to get some coins", user.name, internal.EMOJI_DONT_KNOW)
	if coins := user.coins; coins > 0 {
		msg = fmt.Sprintf("%s, you have %d %s", user.name, internal.EMOJI_COIN)
	}

	gBot.Send(tgbotapi.NewMessage(gChatId, msg))

	showMenu()
}

func callbackQueryIsMissing(update *tgbotapi.Update) bool {
	return update.CallbackQuery == nil || update.CallbackQuery.From == nil
}

func getUserFromUpdate(update *tgbotapi.Update) (user *User, found bool) {
	if callbackQueryIsMissing(update) {
		return
	}

	userId := update.CallbackQuery.From.ID
	for _, userInChat := range gUsersInChat {
		if userId == userInChat.id {
			return userInChat, true
		}
	}
	return
}

func saveUserFromUpdate(update *tgbotapi.Update) (user *User, found bool) {
	if callbackQueryIsMissing(update) {
		return
	}

	from := update.CallbackQuery.From

	user = &User{
		id:    from.ID,
		name:  strings.TrimSpace(from.FirstName + " " + from.LastName),
		coins: 0,
	}

	gUsersInChat = append(gUsersInChat, user)
	return user, true
}

func showActivities(activities Activities, text string, isUsefull bool) {
	activitiesButtonRows := make([]([]tgbotapi.InlineKeyboardButton), 0, len(activities)+1)
	for _, activity := range activities {
		activityDescription := ""
		if isUsefull {
			activityDescription = fmt.Sprintf("+ %d %s: %s", activity.coins, internal.EMOJI_COIN, activity.name)
		} else {
			activityDescription = fmt.Sprintf("- %d %s: %s", activity.coins, internal.EMOJI_COIN, activity.name)
		}

		activitiesButtonRows = append(activitiesButtonRows, getKeyboardRow(activityDescription, activity.code))
	}
	activitiesButtonRows = append(activitiesButtonRows, getKeyboardRow(internal.BUTTON_TEXT_PRINT_MENU, internal.BUTTON_CODE_PRINT_MENU))

	msg := tgbotapi.NewMessage(gChatId, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(activitiesButtonRows...)
	gBot.Send(msg)
}

func showUsefulActivities() {
	showActivities(gUsefullActivities, "Options to track your activity", true)
}

func showRewards() {
	showActivities(gRewards, "Pick your reward", false)
}

func findActivity(activities Activities, choiseCode string) (fActivity *Activity, found bool) {
	for _, rActivity := range activities {
		if choiseCode == rActivity.code {
			return rActivity, true
		}
	}
	return nil, false
}

func processUsefullActivity(usefullActivity *Activity, user *User) {
	errorMsg := ""

	if usefullActivity.coins == 0 {
		errorMsg = fmt.Sprintf(`the activity "%s" doesn't have a specified cost`, usefullActivity.name)
	} else if user.coins > internal.MaxUserCoins {
		errorMsg = fmt.Sprintf("you cannot have more than %d %s", internal.MaxUserCoins, internal.EMOJI_COIN)
	}

	resultMessage := ""
	if errorMsg != "" {
		resultMessage = fmt.Sprintf("%s, I'm sorry, but %s %s Your balance remains unchanged.", user.name, errorMsg, internal.EMOJI_SAD)
	} else {
		user.coins += usefullActivity.coins
		resultMessage = fmt.Sprintf(`%s, the "%s" activity is completed! %d %s has been added to your account. Keep it up! %s%s Now you have %d %s`,
			user.name, usefullActivity.name, usefullActivity.coins, internal.EMOJI_COIN, internal.EMOJI_BICEPS, internal.EMOJI_SUNGLASSES, user.coins, internal.EMOJI_COIN)
	}

	gBot.Send(tgbotapi.NewMessage(gChatId, resultMessage))
}

func processReward(reward *Activity, user *User) {
	errorMsg := ""
	if reward.coins == 0 {
		errorMsg = fmt.Sprintf(`the reward "%s" doesn't have a specified cost`, reward.name)
	} else if user.coins < reward.coins {
		errorMsg = fmt.Sprintf(`you currently have %d %s. You cannot afford "%s" for %d %s`, user.coins, internal.EMOJI_COIN, reward.name, reward.coins, internal.EMOJI_COIN)
	}

	resultMessage := ""
	if errorMsg != "" {
		resultMessage = fmt.Sprintf("%s, I'm sorry, but %s %s Your balance remains unchanged, the reward is unavailable %s",
			user.name, errorMsg, internal.EMOJI_SAD, internal.EMOJI_DONT_KNOW)
	} else {
		user.coins -= reward.coins
		resultMessage = fmt.Sprintf(`%s, the reward "%s" has been paid for, get started! %d %s has been deducted from your account. Now you have %d %s`,
			user.name, reward.name, reward.coins, internal.EMOJI_COIN, user.coins, internal.EMOJI_COIN)
	}
	gBot.Send(tgbotapi.NewMessage(gChatId, resultMessage))
}

func updateProcessing(update *tgbotapi.Update) {
	user, found := getUserFromUpdate(update)
	if !found {
		if user, found = saveUserFromUpdate(update); !found {
			gBot.Send(tgbotapi.NewMessage(gChatId, "User identification failed"))
			return
		}
	}
	choiseCode := update.CallbackQuery.Data
	log.Printf("[&T] %s", time.Now(), choiseCode)

	switch choiseCode {
	case internal.BUTTON_CODE_BALANCE:
		showBalance(user)
	case internal.BUTTON_CODE_USEFUL_ACTIVITIES:
		showUsefulActivities()
	case internal.BUTTON_CODE_REWARDS:
		showRewards()
	case internal.BUTTON_CODE_PRINT_INTRO:
		printIntro(update)
		showMenu()
	case internal.BUTTON_CODE_SKIP_INTRO:
		showMenu()
	case internal.BUTTON_CODE_PRINT_MENU:
		showMenu()
	default:
		if usefullActivity, found := findActivity(gUsefullActivities, choiseCode); found {
			processUsefullActivity(usefullActivity, user)

			delay(2)

			showUsefulActivities()
			return
		}

		if reward, found := findActivity(gRewards, choiseCode); found {
			processReward(reward, user)

			delay(2)

			showUsefulActivities()
			return
		}
		log.Printf("[%T] !!! Error: Unknown code %s", time.Now(), choiseCode)
		msg := fmt.Sprintf("%s, I'm sorry, I don't recognize code '%s' %s Please report this error to my creator.", user.name, choiseCode, internal.EMOJI_SAD)
		gBot.Send(tgbotapi.NewMessage(gChatId, msg))

	}

}

func isStartMessage(update *tgbotapi.Update) bool {
	return update.Message != nil && update.Message.Text == "/start"
}

func delay(seconds uint8) {
	time.Sleep(time.Second * time.Duration(seconds))
}

func sendMessageWithDelay(delayInSec uint8, message string) {
	gBot.Send(tgbotapi.NewMessage(gChatId, message))
	delay(delayInSec)
}

func printIntro(update *tgbotapi.Update) {
	sendMessageWithDelay(2, "Hello! "+internal.EMOJI_SUNGLASSES)
	sendMessageWithDelay(7, "There are numerous beneficial actions that, by performing regularly, we improve the quality of our life. However, often it's more fun, easier, or tastier to do something harmful. Isn't that so?")
	sendMessageWithDelay(7, "With greater likelihood, we'll prefer to get lost in YouTube Shorts instead of an English lesson, buy M&M's instead of vegetables, or lie in bed instead of doing yoga.")
	sendMessageWithDelay(1, internal.EMOJI_SAD)
	sendMessageWithDelay(10, "Everyone has played at least one game where you need to level up a character, making them stronger, smarter, or more beautiful. It's enjoyable because each action brings results. In real life, though, systematic actions over time start to become noticeable. Let's change that, shall we?")
	sendMessageWithDelay(1, internal.EMOJI_SMILE)
	sendMessageWithDelay(14, `Before you are two tables: "Useful Activities" and "Rewards". The first table lists simple short activities, and for completing each of them, you'll earn the specified amount of coins. In the second table, you'll see a list of activities you can only do after paying for them with coins earned in the previous step.`)
	sendMessageWithDelay(1, internal.EMOJI_COIN)
	sendMessageWithDelay(10, `For example, you spend half an hour doing yoga, for which you get 2 coins. After that, you have 2 hours of programming study, for which you get 8 coins. Now you can watch 1 episode of "Interns" and break even. It's that simple!`)
	sendMessageWithDelay(6, `Mark completed useful activities to not lose your coins. And don't forget to "purchase" the reward before actually doing it.`)
}

func getKeyboardRow(buttonText, buttonCode string) []tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(buttonText, buttonCode))
}

func askToPrintIntro() {
	msg := tgbotapi.NewMessage(gChatId, "Do you want to get my intro?")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		getKeyboardRow(internal.BUTTON_TEXT_PRINT_INTRO, internal.BUTTON_CODE_PRINT_INTRO),
		getKeyboardRow(internal.BUTTON_TEXT_SKIP_INTRO, internal.BUTTON_CODE_SKIP_INTRO),
	)

	gBot.Send(msg)
}

func run() (err error) {
	log.Printf("Authorized on account %s", gBot.Self.UserName)

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = internal.UpdateConfigTimeout

	for update := range gBot.GetUpdatesChan(updateConfig) {
		if isCallbackQueryNil(&update) {
			updateProcessing(&update)
		} else if isStartMessage(&update) {
			log.Printf("Reply: [%s] %s", update.Message.From.UserName, update.Message.Text)
			gChatId = update.Message.Chat.ID

			askToPrintIntro()
		}

	}

	return err
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		log.Fatal(err)
	}
}
