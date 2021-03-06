package jwtsecrets

import (
	"context"
	"regexp"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	keyRotationDuration    = "key_ttl"
	keyTokenTTL            = "jwt_ttl"
	keySetIAT              = "set_iat"
	keySetJTI              = "set_jti"
	keySetNBF              = "set_nbf"
	keyIssuer              = "issuer"
	keyAudiencePattern     = "audience_pattern"
	keySubjectPattern      = "subject_pattern"
	keyMaxAllowedAudiences = "max_audiences"
	keyAllowedClaims       = "allowed_claims"
)

func pathConfig(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "config",
		Fields: map[string]*framework.FieldSchema{
			keyRotationDuration: {
				Type:        framework.TypeString,
				Description: `Duration before a key stops being used to sign new tokens.`,
			},
			keyTokenTTL: {
				Type:        framework.TypeString,
				Description: `Duration a token is valid for.`,
			},
			keySetIAT: {
				Type:        framework.TypeBool,
				Description: `Whether or not the backend should generate and set the 'iat' claim.`,
			},
			keySetJTI: {
				Type:        framework.TypeBool,
				Description: `Whether or not the backend should generate and set the 'jti' claim.`,
			},
			keySetNBF: {
				Type:        framework.TypeBool,
				Description: `Whether or not the backend should generate and set the 'nbf' claim.`,
			},
			keyIssuer: {
				Type:        framework.TypeString,
				Description: `Value to set as the 'iss' claim. Claim is omitted if empty.`,
			},
			keyAudiencePattern: {
				Type:        framework.TypeString,
				Description: `Regular expression which must match incoming 'aud' claims.`,
			},
			keySubjectPattern: {
				Type:        framework.TypeString,
				Description: `Regular expression which must match incoming 'sub' claims`,
			},
			keyMaxAllowedAudiences: {
				Type:        framework.TypeInt,
				Description: `Maximum number of allowed audiences, or -1 for no limit.`,
			},
			keyAllowedClaims: {
				Type: framework.TypeStringSlice,
				Description: `Claims which are able to be set in addition to ones generated by the backend.
Note: 'aud' and 'sub' should be in this list if you would like to set them.`,
			},
		},

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathConfigWrite,
			},
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathConfigRead,
			},
		},

		HelpSynopsis:    pathConfigHelpSyn,
		HelpDescription: pathConfigHelpDesc,
	}
}

func (b *backend) pathConfigWrite(c context.Context, r *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	b.configLock.Lock()
	defer b.configLock.Unlock()

	if newRotationPeriod, ok := d.GetOk(keyRotationDuration); ok {
		duration, err := time.ParseDuration(newRotationPeriod.(string))
		if err != nil {
			return nil, err
		}
		b.config.KeyRotationPeriod = duration
	}

	if newTTL, ok := d.GetOk(keyTokenTTL); ok {
		duration, err := time.ParseDuration(newTTL.(string))
		if err != nil {
			return nil, err
		}
		b.config.TokenTTL = duration
	}

	if newSetIat, ok := d.GetOk(keySetIAT); ok {
		b.config.SetIAT = newSetIat.(bool)
	}

	if newSetJTI, ok := d.GetOk(keySetJTI); ok {
		b.config.SetJTI = newSetJTI.(bool)
	}

	if newSetNBF, ok := d.GetOk(keySetNBF); ok {
		b.config.SetNBF = newSetNBF.(bool)
	}

	if newIssuer, ok := d.GetOk(keyIssuer); ok {
		b.config.Issuer = newIssuer.(string)
	}

	if newAudiencePattern, ok := d.GetOk(keyAudiencePattern); ok {
		pattern, err := regexp.Compile(newAudiencePattern.(string))
		if err != nil {
			return nil, err
		}
		b.config.AudiencePattern = pattern
	}

	if newSubjectPattern, ok := d.GetOk(keySubjectPattern); ok {
		pattern, err := regexp.Compile(newSubjectPattern.(string))
		if err != nil {
			return nil, err
		}
		b.config.SubjectPattern = pattern
	}

	if newMaxAudiences, ok := d.GetOk(keyMaxAllowedAudiences); ok {
		b.config.MaxAudiences = newMaxAudiences.(int)
	}

	if newAllowedClaims, ok := d.GetOk(keyAllowedClaims); ok {
		b.config.AllowedClaims = newAllowedClaims.([]string)
		b.config.allowedClaimsMap = makeAllowedClaimsMap(newAllowedClaims.([]string))
	}

	return nonLockingRead(b)
}

func (b *backend) pathConfigRead(_ context.Context, _ *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	b.configLock.RLock()
	defer b.configLock.RUnlock()

	return nonLockingRead(b)
}

func nonLockingRead(b *backend) (*logical.Response, error) {
	return &logical.Response{
		Data: map[string]interface{}{
			keyRotationDuration:    b.config.KeyRotationPeriod.String(),
			keyTokenTTL:            b.config.TokenTTL.String(),
			keySetIAT:              b.config.SetIAT,
			keySetJTI:              b.config.SetJTI,
			keySetNBF:              b.config.SetNBF,
			keyIssuer:              b.config.Issuer,
			keyAudiencePattern:     b.config.AudiencePattern.String(),
			keySubjectPattern:      b.config.SubjectPattern.String(),
			keyMaxAllowedAudiences: b.config.MaxAudiences,
			keyAllowedClaims:       b.config.AllowedClaims,
		},
	}, nil
}

const pathConfigHelpSyn = `
Configure the backend.
`

const pathConfigHelpDesc = `
Configure the backend.

key_ttl:          Duration before a key stops signing new tokens and a new one is generated.
		          After this period the public key will still be available to verify JWTs.
jwt_ttl:          Duration before a token expires.
set_iat:          Whether or not the backend should generate and set the 'iat' claim.
set_jti:          Whether or not the backend should generate and set the 'jti' claim.
set_nbf:          Whether or not the backend should generate and set the 'nbf' claim.
issuer:           Value to set as the 'iss' claim. Claim omitted if empty.
audience_pattern: Regular expression which must match incoming 'aud' claims.
subject_pattern:  Regular expression which must match incoming 'sub' claims.
max_audiences:    Maximum number of allowed audiences, or -1 for no limit.
allowed_claims:   Claims which are able to be set in addition to ones generated by the backend.
                  Note: 'aud' and 'sub' should be in this list if you would like to set them.
`
