package main

import (
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vvampirius/bibExchangesBot/nextDate"
	"github.com/vvampirius/mygolibs/belinvestbankExchange"
	"github.com/vvampirius/mygolibs/telegram"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
	"time"
)

type Store struct {
	LastCheck CurrencyCheck
	Chats     map[int]Chat
}

type CurrencyCheck struct {
	Time time.Time
	Value float64
}

type Chat struct {
	Id int
	Raise bool
	Fall bool
}

type Core struct {
	LastCheck *CurrencyCheck
	Chats map[int]Chat
	StoragePath string
	Me telegram.Me
	Token string
	saveMu sync.Mutex
	priceGauge prometheus.Gauge
	getCurrenciesError prometheus.Counter
	sendMessageError prometheus.Counter
	commandsCount prometheus.Counter
	salaryNotifyTimer *time.Timer
}

func (core *Core) Raise(last, current CurrencyCheck) {
	msg := fmt.Sprintf("Курс доллара вырос с %.3f до %.3f. 🔺", last.Value, current.Value)
	log.Println(msg)
	for _, chat := range core.Chats {
		if !chat.Raise { continue }
		log.Printf("Notify %d chat\n", chat.Id)
		if err := telegram.SendMessage(core.Token, chat.Id, msg, false, 0); err != nil {
			if err.(*telegram.SendMessageError).ErrCode == 403 {
				log.Printf("Removing %d from chats\n", chat.Id)
				delete(core.Chats, chat.Id)
				go core.Save()
			} else { core.sendMessageError.Inc() }
		}
	}
}

func (core *Core) Fall(last, current CurrencyCheck) {
	msg := fmt.Sprintf("Курс доллара упал с %.3f до %.3f. 🔻", last.Value, current.Value)
	log.Println(msg)
	for _, chat := range core.Chats {
		if !chat.Fall { continue }
		log.Printf("Notify %d chat\n", chat.Id)
		if err := telegram.SendMessage(core.Token, chat.Id, msg, false, 0); err != nil {
			if err.(*telegram.SendMessageError).ErrCode == 403 {
				log.Printf("Removing %d from chats\n", chat.Id)
				delete(core.Chats, chat.Id)
				go core.Save()
			} else { core.sendMessageError.Inc() }
		}
	}
}

func (core *Core) checkRoutine() {
	for {
		if currency, err := core.getCurrencyCheck(); err == nil {
			log.Printf("Got exchange rate: %g\n", currency.Value)
			if core.LastCheck != nil && currency.Value > core.LastCheck.Value { go core.Raise(*core.LastCheck, currency)}
			if core.LastCheck != nil && currency.Value < core.LastCheck.Value { go core.Fall(*core.LastCheck, currency)}
			core.LastCheck = &currency
			core.priceGauge.Set(currency.Value)
			core.Save()
		} else { core.getCurrenciesError.Inc() }
		time.Sleep(20 * time.Minute)
	}
}

func (core *Core) getCurrencyCheck() (CurrencyCheck, error) {
	//for development only!
	//f, _ := os.Open(path.Join(core.StoragePath, `rates.html`))
	//defer f.Close()
	//---------------------------

	currencies, err := belinvestbankExchange.Get(nil)
	if err != nil { return CurrencyCheck{}, err	}
	usd, ok := currencies[`USD`]
	if !ok {
		err := errors.New(`USD currency not found`)
		log.Println(err.Error())
		return CurrencyCheck{}, err
	}
	return CurrencyCheck{time.Now(), usd.Sell}, nil
}

func (core *Core) SetChat(chatId int, raise, fall bool) {
	chat := Chat{
		Id: chatId,
		Raise: raise,
		Fall: fall,
	}
	core.Chats[chatId] = chat
}

func (core *Core) httpHandler(wr http.ResponseWriter, request *http.Request) {
	log.Println(request.Method, request.RequestURI)
	log.Println(request.Header)
	if request.Method != http.MethodPost { return }
	data, err := ioutil.ReadAll(request.Body)
	if err != nil {
		log.Println(err.Error())
		return
	}
	log.Println(string(data))

	update, err := telegram.UnmarshalUpdate(data)
	if err != nil { return }
	log.Println(update)

	if update.Message.IsBotCommand() {
		core.commandsCount.Inc()
		chatId := update.Message.Chat.Id
		switch update.Message.Text {
		case `/start`:
			log.Println(`start command`)
			core.SetChat(chatId, false, false)
			go telegram.SendMessage(core.Token, chatId, `Привет! Смотри команды, что бы активировать необходимые оповещения! 😉`, false, 0)
		case `/all`:
			log.Println(`all command`)
			core.SetChat(chatId, true, true)
			go telegram.SendMessage(core.Token, chatId, `👍 Буду слать тебе оповещения о любых изменениях курса.`, false, 0)
		case `/raise`:
			log.Println(`raise command`)
			core.SetChat(chatId, true, false)
			go telegram.SendMessage(core.Token, chatId, `Ок! Буду слать тебе оповещения только о повышении курса. 🔺`, false, 0)
		case `/fall`:
			log.Println(`fall command`)
			core.SetChat(chatId, false, true)
			go telegram.SendMessage(core.Token, chatId, `Ок! Буду слать тебе оповещения только о снижении курса. 🔻`, false, 0)
		case `/none`:
			log.Println(`none command`)
			core.SetChat(chatId, false, false)
			go telegram.SendMessage(core.Token, chatId, `Ок! Не буду слать тебе оповещений о изменении курса. 🌙`, false, 0)
		}
		core.Save()
	}
}

func (core *Core) Save() error {
	core.saveMu.Lock()
	defer core.saveMu.Unlock()

	f, err := os.Create(path.Join(core.StoragePath, `store.gob`))
	if err != nil {
		log.Println(err.Error())
		return err
	}
	defer f.Close()

	store := Store{
		LastCheck: CurrencyCheck{},
		Chats: core.Chats,
	}
	if core.LastCheck != nil { store.LastCheck = *core.LastCheck }

	encoder := gob.NewEncoder(f)
	if err := encoder.Encode(store); err != nil {
		log.Println(store, err.Error())
		return err
	}
	return nil
}

func (core *Core) Load() error {
	f, err := os.Open(path.Join(core.StoragePath, `store.gob`))
	if err != nil {
		log.Println(err.Error())
		return err
	}
	defer f.Close()

	store := Store{}
	decoder := gob.NewDecoder(f)
	if err := decoder.Decode(&store); err != nil {
		log.Println(err.Error())
		return err
	}

	core.Chats = store.Chats
	if !store.LastCheck.Time.IsZero() { core.LastCheck = &store.LastCheck }
	return nil
}

func (core *Core) nearSalaryNotifyAfter() time.Duration {
	notifyDate := nextDate.NearMonthlyDate(time.Now(), 10, `13:00:00`, true)
	notifyAfter := notifyDate.Sub(time.Now())
	log.Printf("Salary notification will be at '%v' (after %v).\n", notifyDate, notifyAfter)
	return notifyAfter
}

func (core *Core) salaryNotifyRoutine() {
	msg := `💰💰💰 Похоже сегодня день зарплаты, и тебе вероятно опять стоит следить за курсом. 😉`
	for {
		<- core.salaryNotifyTimer.C
		log.Println(`Sending notification about salary...`)
		for _, chat := range core.Chats {
			if chat.Raise && chat.Fall { continue }
			log.Printf("Notify %d chat\n", chat.Id)
			if err := telegram.SendMessage(core.Token, chat.Id, msg, false, 0); err != nil {
				if err.(*telegram.SendMessageError).ErrCode == 403 {
					log.Printf("Removing %d from chats\n", chat.Id)
					delete(core.Chats, chat.Id)
					go core.Save()
				} else { core.sendMessageError.Inc() }
			}
		}
		core.salaryNotifyTimer = time.NewTimer(core.nearSalaryNotifyAfter())
	}
}


func NewCore(storagePath, token, callbackUrl string) (*Core, error) {
	priceGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "price",
		Help: "USD price",
	})
	if err := prometheus.Register(priceGauge); err != nil { log.Fatalln(err.Error()) }

	getCurrenciesError := prometheus.NewCounter(prometheus.CounterOpts{
		Name: `getCurrenciesError`,
		Help: `Error counter of get currencies from belinvestbank.by`,
	})
	if err := prometheus.Register(getCurrenciesError); err != nil { log.Fatalln(err.Error()) }

	sendMessageError := prometheus.NewCounter(prometheus.CounterOpts{
		Name: `sendMessageError`,
		Help: `Error counter of sending messages to Telegram`,
	})
	if err := prometheus.Register(sendMessageError); err != nil { log.Fatalln(err.Error()) }

	commandsCount := prometheus.NewCounter(prometheus.CounterOpts{
		Name: `commands`,
		Help: `Commands counter`,
	})
	if err := prometheus.Register(commandsCount); err != nil { log.Fatalln(err.Error()) }

	me, err := telegram.GetMe(token)
	if err != nil { return nil, err }
	log.Printf("Got info from Telegram API: @%s with ID:%d and name '%s'\n", me.Username, me.Id, me.FirstName)

	if err := telegram.SetWebHook(token, callbackUrl); err != nil { return nil, err }
	log.Printf("Callback URL set to '%s'\n", callbackUrl)

	core := Core{
		Chats: make(map[int]Chat),
		StoragePath: storagePath,
		Me: me,
		Token: token,
		priceGauge: priceGauge,
		getCurrenciesError: getCurrenciesError,
		sendMessageError: sendMessageError,
		commandsCount: commandsCount,
	}
	core.Load()

	chatsCount := prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: `chats`,
		Help: `Number of bot consumers`,
	}, func() float64 { return float64(len(core.Chats)) })
	if err := prometheus.Register(chatsCount); err != nil { log.Fatalln(err.Error()) }

	//TODO: get pull updates

	go core.checkRoutine()

	core.salaryNotifyTimer = time.NewTimer(core.nearSalaryNotifyAfter())
	go core.salaryNotifyRoutine()

	return &core, nil
}
