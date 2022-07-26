package authentication

type Store struct {
	store map[string]bool
}

func NewStore() Store {
	return Store{map[string]bool{}}
}

func (st Store) TokenExists(token string) bool {
	_, exists := st.store[token]
	return exists
}

func (st Store) StoreToken(token string) {
	st.store[token] = true
}

func (st Store) RemoveToken(token string) bool {
	_, exists := st.store[token]

	if !exists {
		return false
	}

	delete(st.store, token)
	return true
}
