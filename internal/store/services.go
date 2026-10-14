package store

import (
	"math/rand"
	"time"
)

var realms = []struct {
	Name      string
	Threshold int
}{{`炼气期`, 100}, {`筑基期`, 180}, {`金丹期`, 280}, {`元婴期`, 400}, {`化神期`, 560}, {`炼虚期`, 760}, {`合体期`, 1000}, {`大乘期`, 1300}, {`渡劫期`, 0}}

func (s *Store) saveUser(id, n string) error {
	_, e := s.DB.Exec(`INSERT INTO users(user_id,nickname) VALUES(?,?) ON CONFLICT(user_id) DO UPDATE SET nickname=excluded.nickname,updated_at=CURRENT_TIMESTAMP`, id, n)
	return e
}
func (s *Store) SignIn(id, n string) (bool, int, int, error) {
	if e := s.saveUser(id, n); e != nil {
		return false, 0, 0, e
	}
	d := time.Now().Format(`2006-01-02`)
	var c int
	if e := s.DB.QueryRow(`SELECT COUNT(*) FROM sign_ins WHERE user_id=? AND sign_date=?`, id, d).Scan(&c); e != nil {
		return false, 0, 0, e
	}
	if c > 0 {
		return false, 0, 0, nil
	}
	r := rand.Intn(16) + 5
	result, e := s.DB.Exec(`INSERT INTO sign_ins(user_id,sign_date) VALUES(?,?) ON CONFLICT(user_id,sign_date) DO NOTHING`, id, d)
	if e != nil {
		return false, 0, 0, e
	}
	inserted, e := result.RowsAffected()
	if e != nil {
		return false, 0, 0, e
	}
	if inserted == 0 {
		return false, 0, 0, nil
	}
	e = s.DB.QueryRow(`SELECT COUNT(*) FROM sign_ins WHERE user_id=?`, id).Scan(&c)
	return e == nil, r, c, e
}

type Profile struct {
	Name                          string
	Experience, Threshold, Stones int
}

func (s *Store) profile(id, n string) (Profile, error) {
	_, e := s.DB.Exec(`INSERT INTO cultivators(user_id,nickname) VALUES(?,?) ON CONFLICT(user_id) DO UPDATE SET nickname=excluded.nickname`, id, n)
	if e != nil {
		return Profile{}, e
	}
	var i int
	p := Profile{}
	e = s.DB.QueryRow(`SELECT realm_index,experience,spirit_stones FROM cultivators WHERE user_id=?`, id).Scan(&i, &p.Experience, &p.Stones)
	if e == nil {
		p.Name = realms[i].Name
		p.Threshold = realms[i].Threshold
	}
	return p, e
}
func (s *Store) Profile(id, n string) (Profile, error) { return s.profile(id, n) }
func (s *Store) Cultivate(id, n string) (Profile, int, error) {
	p, e := s.profile(id, n)
	if e != nil {
		return p, 0, e
	}
	g := rand.Intn(17) + 8
	_, e = s.DB.Exec(`UPDATE cultivators SET experience=experience+?,spirit_stones=spirit_stones+1,updated_at=CURRENT_TIMESTAMP WHERE user_id=?`, g, id)
	if e != nil {
		return p, 0, e
	}
	p, e = s.profile(id, n)
	return p, g, e
}
func (s *Store) Breakthrough(id, n string) (string, string, bool, bool, Profile, error) {
	p, e := s.profile(id, n)
	if e != nil {
		return ``, ``, false, false, p, e
	}
	var i int
	e = s.DB.QueryRow(`SELECT realm_index FROM cultivators WHERE user_id=?`, id).Scan(&i)
	if i >= len(realms)-1 {
		return ``, ``, false, true, p, e
	}
	if p.Experience < realms[i].Threshold {
		return ``, ``, false, false, p, e
	}
	old := realms[i].Name
	i++
	now := realms[i].Name
	_, e = s.DB.Exec(`UPDATE cultivators SET realm_index=?,experience=0,updated_at=CURRENT_TIMESTAMP WHERE user_id=?`, i, id)
	return old, now, e == nil, false, p, e
}
