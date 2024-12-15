# 17. use plausible.io to collect statistics

Date: 2024-04-14

## Status

Accepted

## Context

A fundamental requirement is collecting usage statistics for various arc42 website, with high accuracy. Spam traffic shall be excluded. 
Compliant with EU privacy regulations.
Accurate results over all websites.

## Decision

It's perfectly possible to build a website and visitor counter, but the effort to get both reliability and privacy right seemed overly high.
Recognizing spam traffic might be very difficult.

Therefore, we decided to use plausible.io, a commercial offering.
From their website:
>Plausible is intuitive, lightweight and open source web analytics. No cookies and fully compliant with GDPR, CCPA and PECR. Made and hosted in the EU, powered by European-owned cloud infrastructure

## Consequences

Just a small JavaScript snippet has to be included in the header of all pages that needs to be included in counting. Jekyll makes this fairly easy.