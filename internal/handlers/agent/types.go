package agent

// RegistrationRequest represents the data received from an agent during registration
type RegistrationRequest struct {
	ClaimCode string `json:"claim_code"`
	Hostname  string `json:"hostname"`
}

// RegistrationResponse represents the data sent back to the agent after successful registration
type RegistrationResponse struct {
	AgentID       int               `json:"agent_id"`
	DownloadToken string            `json:"download_token"`
	Endpoints     map[string]string `json:"endpoints"`
}

// CertificateResponse represents the certificate data sent to the agent
type CertificateResponse struct {
	AgentID       int    `json:"agent_id"`
	Certificate   string `json:"certificate"`
	PrivateKey    string `json:"private_key"`
	CACertificate string `json:"ca_certificate"`
}
