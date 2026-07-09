package service

import (
	"sort"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"
)

func appendUniqueTag(tags []string, tag string) []string {
	for _, existing := range tags {
		if existing == tag {
			return tags
		}
	}
	return append(tags, tag)
}

func (s *InboundService) LocalClientInboundTags() (map[string][]string, error) {
	db := database.GetDB()
	var inbounds []model.Inbound
	if err := db.Where("node_id IS NULL AND enable = ?", true).Find(&inbounds).Error; err != nil {
		return nil, err
	}
	emailTags := make(map[string][]string)
	for i := range inbounds {
		ib := &inbounds[i]
		tag := strings.TrimSpace(ib.Tag)
		if tag == "" {
			continue
		}
		clients, err := s.GetClients(ib)
		if err != nil {
			continue
		}
		for _, client := range clients {
			email := strings.TrimSpace(client.Email)
			if email == "" || !client.Enable {
				continue
			}
			emailTags[email] = appendUniqueTag(emailTags[email], tag)
		}
	}
	return emailTags, nil
}

func resolveClientSessionTags(
	emailInboundTags map[string][]string,
	tagDelta map[string]int64,
	pollActiveTags map[string]struct{},
	activeEmails []string,
	previous map[string]string,
) map[string]string {
	if len(emailInboundTags) == 0 || len(activeEmails) == 0 {
		return nil
	}
	out := make(map[string]string, len(activeEmails))
	for _, email := range activeEmails {
		if email == "" {
			continue
		}
		tags := emailInboundTags[email]
		if len(tags) == 0 {
			continue
		}
		var withDelta []string
		for _, tag := range tags {
			if tagDelta[tag] > 0 {
				withDelta = append(withDelta, tag)
			}
		}
		if len(withDelta) == 1 {
			out[email] = withDelta[0]
			continue
		}
		if len(withDelta) > 1 {
			best := withDelta[0]
			bestDelta := tagDelta[best]
			for _, tag := range withDelta[1:] {
				if tagDelta[tag] > bestDelta {
					best = tag
					bestDelta = tagDelta[tag]
				}
			}
			out[email] = best
			continue
		}
		var activeCand []string
		for _, tag := range tags {
			if _, ok := pollActiveTags[tag]; ok {
				activeCand = append(activeCand, tag)
			}
		}
		if len(activeCand) == 1 {
			out[email] = activeCand[0]
			continue
		}
		if prev, ok := previous[email]; ok {
			for _, tag := range tags {
				if tag == prev {
					out[email] = prev
					break
				}
			}
			if out[email] != "" {
				continue
			}
		}
		if len(tags) == 1 {
			out[email] = tags[0]
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (s *InboundService) PollSessionTags(
	traffics []*xray.Traffic,
	clientTraffics []*xray.ClientTraffic,
	connectionOnlines []string,
) map[string]string {
	emailTags, err := s.LocalClientInboundTags()
	if err != nil || len(emailTags) == 0 {
		return nil
	}
	tagDelta := make(map[string]int64)
	pollActive := make(map[string]struct{})
	for _, tr := range traffics {
		if tr == nil || !tr.IsInbound {
			continue
		}
		delta := tr.Up + tr.Down
		tagDelta[tr.Tag] = delta
		if delta > 0 {
			pollActive[tr.Tag] = struct{}{}
		}
	}
	activeSet := make(map[string]struct{})
	for _, ct := range clientTraffics {
		if ct != nil && ct.Up+ct.Down > 0 {
			activeSet[ct.Email] = struct{}{}
		}
	}
	for _, email := range connectionOnlines {
		if email != "" {
			activeSet[email] = struct{}{}
		}
	}
	emails := make([]string, 0, len(activeSet))
	for email := range activeSet {
		emails = append(emails, email)
	}
	sort.Strings(emails)
	var previous map[string]string
	if p != nil {
		previous = p.GetLocalClientSessionTags()
	}
	return resolveClientSessionTags(emailTags, tagDelta, pollActive, emails, previous)
}

func (s *InboundService) GetClientSessionTagsByGuid() map[string]map[string]string {
	if p == nil {
		return map[string]map[string]string{}
	}
	tags := p.GetLocalClientSessionTags()
	if len(tags) == 0 {
		return map[string]map[string]string{}
	}
	guid := s.panelGuid()
	if guid == "" {
		return map[string]map[string]string{}
	}
	return map[string]map[string]string{guid: tags}
}
