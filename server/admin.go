package server

type admin struct {
	qm *queuemanager
}

func (a *admin) SetToken(app, tokentype, token string) error {
	q, err := a.qm.Get("admin")
	if err != nil {
		return err
	}

	if len(token) == 0 {
		q.Delete(app, tokentype)
	}

	return q.Set(app, tokentype, token)
}

func (a *admin) GetToken(app, tokentype string) (string, error) {
	q, err := a.qm.Get("admin")
	if err != nil {
		return "", err
	}

	return q.Get(app, tokentype)
}

func newadmin(qm *queuemanager) *admin {
	return &admin{
		qm: qm,
	}
}
