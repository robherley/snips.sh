package msgs

type Error struct {
	Err error
}

func (em Error) Error() string {
	return em.Err.Error()
}
