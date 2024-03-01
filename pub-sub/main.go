package main

type Message struct {
	Content string
	Topic   string
}

type Publisher struct {
	c           chan Message
	Subscribers map[string][]Subscriber
}

func NewPublisher() *Publisher {
	return &Publisher{
		make(chan Message, 10),
		make(map[string][]Subscriber),
	}
}

func (p *Publisher) Publish(msg Message) {
	for _, sub := range p.Subscribers[msg.Topic] {
		sub.Consume(msg.Content)
	}
}

func (p *Publisher) Register(topic string, s Subscriber) {
	p.Subscribers[topic] = append(p.Subscribers[topic], s)
}

type Subscriber interface {
	Consume(msg string)
}

type EmailSubscriber struct{}

func (s *EmailSubscriber) Consume(msg string) {
	println("Sending email: " + msg)
}

type SMSSubscriber struct{}

func (s *SMSSubscriber) Consume(msg string) {
	println("Sending SMS: " + msg)
}

func main() {
	p := NewPublisher()
	emailSubscriber := &EmailSubscriber{}
	smsSubscriber := &SMSSubscriber{}
	p.Register("email", emailSubscriber)
	p.Register("sms", smsSubscriber)

	p.Publish(Message{"Hello, world!", "email"})
	p.Publish(Message{"Hello, world!", "sms"})
}
