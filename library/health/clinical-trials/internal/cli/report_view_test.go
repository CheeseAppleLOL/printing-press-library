// Copyright 2026 laci141 and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import "testing"

// reportCorpus is deliberately NOT phaseTallyCorpus. That corpus exists to
// separate a per-trial phase tally from a per-entry one and carries no
// Countries at all, so a country assertion written against it would pass no
// matter how the geo loop was broken. The lesson #1971 taught the hard way: a
// fixture where every item has exactly one of something cannot detect a
// per-item multiplicity bug.
//
// So every field this file asserts on appears here at three multiplicities —
// zero, one, and more than one:
//
//	Countries: one trial with none, three with one, two with two or three.
//	Phases:    one trial with none, four with one, one with two.
//	Sponsor:   a scalar, so it can only ever be one per trial. Two trials
//	           share a sponsor so the counter has something to merge.
func reportCorpus() []Trial {
	return []Trial{
		{NCTID: "NCT01", Sponsor: "Acme", Phases: []string{"PHASE1"}, Countries: []string{"United States"}},
		{NCTID: "NCT02", Sponsor: "Acme", Phases: []string{"PHASE2"}, Countries: []string{"United States", "Canada"}},
		{NCTID: "NCT03", Sponsor: "Bolt", Phases: []string{"PHASE2", "PHASE3"}, Countries: []string{"Germany"}},
		{NCTID: "NCT04", Sponsor: "Cedar", Phases: []string{"PHASE3"}, Countries: []string{"Japan", "Canada", "Germany"}},
		{NCTID: "NCT05", Sponsor: "Delta", Phases: []string{"PHASE4"}, Countries: []string{"Canada"}},
		{NCTID: "NCT06", Sponsor: "Echo"}, // observational, no phases, no sites posted
	}
}

// The distribution must account for every phase ENTRY, which is the contract
// phase-tally-counts-every-sampled-trial states. Six trials, seven phase
// entries: five singles, one two-phase trial contributing two, and one
// phaseless trial contributing its N/A.
func TestBuildTopicReportCountsPhaseEntries(t *testing.T) {
	rep := buildTopicReport("topic", 900, 300, reportCorpus(), 10)

	if got, want := tallyTotal(rep.Phases), 7; got != want {
		t.Errorf("phase entries summed to %d, want %d from %d trials", got, want, len(reportCorpus()))
	}
	counts := map[string]int{}
	for _, e := range rep.Phases {
		counts[e.Label] = e.Count
	}
	// Phase 2 holds NCT02 plus half of NCT03; Phase 3 holds NCT04 plus the
	// other half. Collapsing a multi-phase trial to its first phase leaves
	// Phase 3 at one.
	if counts["Phase 2"] != 2 {
		t.Errorf("Phase 2 = %d, want 2", counts["Phase 2"])
	}
	if counts["Phase 3"] != 2 {
		t.Errorf("Phase 3 = %d, want 2", counts["Phase 3"])
	}
	// NCT06 posts no phase and belongs in N/A, the same answer phaseDisplay
	// gives it in the Phase column.
	if counts["N/A"] != 1 {
		t.Errorf("N/A = %d, want 1", counts["N/A"])
	}
}

// Countries is the assertion phaseTallyCorpus could not carry. Eight site
// entries across six trials: one trial posts none, three post one, one posts
// two and one posts three.
func TestBuildTopicReportCountsEverySite(t *testing.T) {
	rep := buildTopicReport("topic", 900, 300, reportCorpus(), 10)

	if got, want := tallyTotal(rep.Countries), 8; got != want {
		t.Errorf("country entries summed to %d, want %d", got, want)
	}
	counts := map[string]int{}
	for _, e := range rep.Countries {
		counts[e.Label] = e.Count
	}
	// Canada appears in three separate trials, so a loop that stopped at a
	// trial's first country would leave it at one.
	if counts["Canada"] != 3 {
		t.Errorf("Canada = %d, want 3", counts["Canada"])
	}
	if counts["Germany"] != 2 {
		t.Errorf("Germany = %d, want 2", counts["Germany"])
	}
}

// Sponsor reads one scalar per trial, so the total can only equal the trial
// count. Asserting it pins the difference in shape from Countries: if someone
// ever gives Trial a Sponsors slice, this number stops matching and says so.
func TestBuildTopicReportCountsOneSponsorPerTrial(t *testing.T) {
	corpus := reportCorpus()
	rep := buildTopicReport("topic", 900, 300, corpus, 10)

	if got, want := tallyTotal(rep.Sponsors), len(corpus); got != want {
		t.Errorf("sponsor entries summed to %d, want one per trial (%d)", got, want)
	}
	for _, e := range rep.Sponsors {
		if e.Label == "Acme" && e.Count != 2 {
			t.Errorf("Acme = %d, want 2", e.Count)
		}
	}
}

// The headline counts describe the whole literature and the sample size
// describes what was fetched. They are three different numbers and the
// briefing prints them side by side, so a caller that passed len(trials) for
// total would look plausible on screen.
func TestBuildTopicReportKeepsSampleSeparateFromTotals(t *testing.T) {
	corpus := reportCorpus()
	rep := buildTopicReport("long covid", 900, 300, corpus, 10)

	if rep.Topic != "long covid" {
		t.Errorf("Topic = %q, want %q", rep.Topic, "long covid")
	}
	if rep.Total != 900 {
		t.Errorf("Total = %d, want 900", rep.Total)
	}
	if rep.Recruiting != 300 {
		t.Errorf("Recruiting = %d, want 300", rep.Recruiting)
	}
	if rep.RecruitingPct != 33 {
		t.Errorf("RecruitingPct = %d, want 33", rep.RecruitingPct)
	}
	if rep.SampleSize != len(corpus) {
		t.Errorf("SampleSize = %d, want %d", rep.SampleSize, len(corpus))
	}
}

// topN bounds the two list sections and not the phase distribution, which
// top(8) fixes independently. Five distinct sponsors truncated to two proves
// the parameter reaches geo and sponsor rather than being ignored.
func TestBuildTopicReportRespectsTopN(t *testing.T) {
	rep := buildTopicReport("topic", 900, 300, reportCorpus(), 2)

	if len(rep.Sponsors) != 2 {
		t.Errorf("Sponsors length = %d, want 2", len(rep.Sponsors))
	}
	if len(rep.Countries) != 2 {
		t.Errorf("Countries length = %d, want 2", len(rep.Countries))
	}
}
