---
title: Privacy policy
description: What the Beaver Backlog software and website do, and do not do, with your data.
---

_Last updated: 2 September 2026_

This policy explains how the Beaver Backlog software ("the software") and the website at [beaverbacklog.com](https://beaverbacklog.com) ("the website") handle personal data. Both are maintained by builtbystef ("the maintainer", "we"). The short version: the software never sends your data anywhere, and the website collects only cookieless, aggregate analytics.

## The software

Beaver Backlog is a command-line tool and a local web UI that store issues as Markdown files inside your own project directory.

- **Everything stays on your machine.** Issues, comments, labels, configuration, and every other piece of data the software touches live in files in your project. The software has no server, no account system, and no database of its own.
- **No telemetry.** The software does not collect usage statistics, crash reports, diagnostics, identifiers, or any other information about you or your environment, and it does not send anything to the maintainer or to any third party.
- **No network access at run time.** The software does not connect to the internet. The web UI listens only on your computer's loopback address (`127.0.0.1`), so it is reachable from your own machine and nowhere else unless you deliberately expose it.
- **Your files are yours.** Because issues are plain files, they go wherever you send them. If you commit them to a Git repository and push it to a hosting provider, or let a coding agent read them, that transfer is governed by your own choices and by the policies of those other services, not by this policy.

The maintainer cannot see, access, or recover anything you store with the software.

### The install scripts

The install scripts served from this website (`install.sh` and `install.ps1`) and from the GitHub repository download the software from GitHub. Running them makes requests to `api.github.com` (to find the latest release) and `github.com` (to fetch the release archive). GitHub receives your IP address and standard request headers as part of those requests, subject to the [GitHub Privacy Statement](https://docs.github.com/en/site-policy/privacy-policies/github-general-privacy-statement). The scripts send nothing to the maintainer. Installing with `go install` instead contacts the Go module proxy and GitHub under the same terms.

## The website

The website is a static site: documentation and a landing page. It has no accounts, no sign-up, no forms, no comments, and no way for you to submit personal data to us.

### Analytics

The website uses [Cloudflare Web Analytics](https://www.cloudflare.com/web-analytics/), a privacy-focused, cookieless analytics service. It is used solely to understand, in aggregate, how many people visit the site and which pages they read.

Cloudflare Web Analytics:

- does not set cookies and stores nothing in your browser;
- does not fingerprint your device or track you across other websites;
- does not store your IP address.

The data it reports to us is aggregate only: page URLs visited, referring site, country (derived from the IP address and then discarded), browser and operating system family, device type, and page performance timings. We cannot identify an individual visitor from it.

The analytics script loads from `static.cloudflareinsights.com`. If you block it with a browser extension or content blocker, the website works exactly the same.

### Hosting and request logs

The website is hosted on [Cloudflare Pages](https://pages.cloudflare.com/) and delivered through Cloudflare's global network. To serve a page and protect the site against abuse, Cloudflare necessarily processes the technical data in every web request: your IP address, the URL requested, the time of the request, and the headers your browser sends, such as the user agent and referrer. Cloudflare keeps this data for a limited time for security and operational purposes, as described in the [Cloudflare Privacy Policy](https://www.cloudflare.com/privacypolicy/). The maintainer does not receive or review these raw logs.

### Cookies and local storage

The website sets no cookies.

The only thing it stores in your browser is your light-or-dark theme preference, if you change it with the theme picker in the footer. That value is held in your browser's `localStorage` under the key `starlight-theme`, is never transmitted anywhere, and can be removed by clearing your site data.

### Fonts and third-party content

Fonts and every other asset the website needs are served from the website itself. No page loads scripts, fonts, or trackers from any third party other than the Cloudflare analytics script described above.

The website links to external sites, chiefly [GitHub](https://github.com/builtbystef/beaver-backlog). Those sites have their own privacy policies, and we are not responsible for how they handle your data.

## Legal basis and data subject rights

Where data protection laws such as the EU and UK General Data Protection Regulation (GDPR) apply, the limited processing described above (analytics and hosting request logs) is carried out on the basis of our legitimate interest in operating, securing, and understanding the use of the website (GDPR Article 6(1)(f)). Because the analytics are aggregate and the request logs are held by Cloudflare rather than by us, no consent banner is required and none is shown.

Depending on where you live, you may have rights to access, correct, delete, restrict, or object to the processing of your personal data, to data portability, and to lodge a complaint with a supervisory authority. Because we hold no data that identifies you, there is nothing for us to return, correct, or delete, but you are welcome to contact us and we will confirm this in writing. For the request logs held by Cloudflare, exercise your rights directly with Cloudflare under its privacy policy.

We do not sell personal data, do not share it with third parties for their own marketing, and do not use it for advertising or profiling. We never have and never will.

## International transfers

Cloudflare operates a worldwide network, so a request to the website may be served from, and its technical data may be processed in, a country other than your own. Cloudflare provides safeguards for such transfers, including standard contractual clauses and participation in the EU-US Data Privacy Framework, as described in its privacy policy.

## Children

The website and software are not directed at children under 16, and we do not knowingly collect personal data from them. As we collect no identifying data at all, there is nothing to remove, but a parent or guardian who has a concern is welcome to contact us.

## Security

The website is served over HTTPS only. The software runs entirely on your machine, so the security of your issue data is the security of your machine and of any repository or service you choose to copy it to.

## Changes to this policy

If the website or software ever starts collecting data it does not collect today, this page will be updated first and the date at the top will change. Continued use of the website after a change means you accept the updated policy. The history of this page is public in the [repository](https://github.com/builtbystef/beaver-backlog/commits/main/site/src/content/docs/privacy.md).

## Contact

Questions about this policy, or requests about your data, can be raised by [opening an issue](https://github.com/builtbystef/beaver-backlog/issues) on GitHub, or by contacting the maintainer through their [GitHub profile](https://github.com/builtbystef) if you would rather not post publicly.
