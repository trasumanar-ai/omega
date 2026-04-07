package registry

import "fmt"

type Firm struct {
	ID        string `json:"id"`
	GovID     string `json:"government_id"`
	Name      string `json:"name"`
	OwnerID   string `json:"owner_id"`
	CreatedAt string `json:"created_at"`
}

type FirmMember struct {
	FirmID   string `json:"firm_id"`
	AgentID  string `json:"agent_id"`
	Role     string `json:"role"`
	JoinedAt string `json:"joined_at"`
}

func (r *Registry) CreateFirm(govID, ownerID, name string) (*Firm, error) {
	id, err := genID()
	if err != nil {
		return nil, err
	}

	_, err = r.db.Exec(
		`INSERT INTO firms (id, government_id, name, owner_id) VALUES (?, ?, ?, ?)`,
		id, govID, name, ownerID,
	)
	if err != nil {
		return nil, fmt.Errorf("create firm: %w", err)
	}

	// Owner is automatically a member
	_, err = r.db.Exec(
		`INSERT INTO firm_members (firm_id, agent_id, role) VALUES (?, ?, 'owner')`,
		id, ownerID,
	)
	if err != nil {
		return nil, fmt.Errorf("add owner to firm: %w", err)
	}

	return &Firm{ID: id, GovID: govID, Name: name, OwnerID: ownerID}, nil
}

func (r *Registry) JoinFirm(firmID, agentID string) error {
	_, err := r.db.Exec(
		`INSERT INTO firm_members (firm_id, agent_id, role) VALUES (?, ?, 'member')`,
		firmID, agentID,
	)
	if err != nil {
		return fmt.Errorf("join firm: %w", err)
	}
	return nil
}

func (r *Registry) ListFirms(govID string) ([]Firm, error) {
	rows, err := r.db.Query(
		`SELECT id, government_id, name, owner_id, created_at
		 FROM firms WHERE government_id = ? ORDER BY created_at`, govID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var firms []Firm
	for rows.Next() {
		var f Firm
		if err := rows.Scan(&f.ID, &f.GovID, &f.Name, &f.OwnerID, &f.CreatedAt); err != nil {
			return nil, err
		}
		firms = append(firms, f)
	}
	return firms, rows.Err()
}

func (r *Registry) FirmMembers(firmID string) ([]FirmMember, error) {
	rows, err := r.db.Query(
		`SELECT firm_id, agent_id, role, joined_at
		 FROM firm_members WHERE firm_id = ? ORDER BY joined_at`, firmID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []FirmMember
	for rows.Next() {
		var m FirmMember
		if err := rows.Scan(&m.FirmID, &m.AgentID, &m.Role, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}
