var documents = [

{
    "id": 0,
    "uri": "adrs/004-parallel-collection-of-time-intervals-from-plausible.html",
    "menu": "null",
    "title": "null",
    "text": " 4. parallel collection of time intervals from plausible Date: 2023-12-01 Status Accepted Context The visitor and pageview counts from plausible.io are collected for three distinct time intervals (7D, 30D, 12M). Each of these require a single API call to plausible.io. These can be handled sequentially or in parallel Goroutines. Decision We call plausible.io in parallel Goroutines for the three time intervals. Consequences We need to implement a func to call plausible with the site and time-interval as parameter. The return value needs to include the time-intervall, therefore we define a (kind-of enum) type for these (ADR-0005). "
},

{
    "id": 1,
    "uri": "adrs/001-record-architecture-decisions.html",
    "menu": "null",
    "title": "null",
    "text": " 1. Record architecture decisions Date: 2023-11-27 Status Accepted Context We need to record the architectural decisions made on this project. Decision We will use Architecture Decision Records, as described by Michael Nygard . Consequences See Michael Nygard's article, linked above. For a lightweight ADR toolset, see Nat Pryce's adr-tools . "
},

{
    "id": 2,
    "uri": "adrs/0006-create-local-svg-badges-in-golang.html",
    "menu": "null",
    "title": "null",
    "text": " 6. create-local-svg-badges-in-golang Date: 2023-12-08 Status Accepted Context The badges showing the number of open issues and bugs for each repository have to be loaded. Sending requests to the external service ( shields.io ) adds a runtime dependency, and potential runtime and energy overhead. Decision Pre-load at least 20 such badges for open-issues and bugs to local storage Make these badges available to static Jekyll site under images/badges/ Add function to create appropriate path/filename combination, so these can be added to HTML output Consequences New golang app in /cmd "
},

{
    "id": 3,
    "uri": "adrs/003-use-arc42statistics-struct-to-collect-results.html",
    "menu": "null",
    "title": "null",
    "text": " 3. use Arc42Statistics struct to collect results Date: 2023-12-01 Status Accepted Context Within the domain package we need to collect the results of the various API calls (to plausible.io and github.com). Decision We use a complex struct ( types.Arc42Statistics ) to collect these results: type Arc42Statistics struct { // some meta info, like AppVersion and time of last update // some info on the server that collected these results (e.g. fly.io region) // Stats4Site contains the statistics per site or subdomain Stats4Site [len(Arc42sites)]SiteStats // Totals contains the sum of all the statistics over all sites Totals TotalsForAllSites } Core of this is Stats4Site , which holds the statistics (visitors and pageviews) for all sites: Stats4Site type SiteStats struct { Site string // site name Repo string // the URL of the GitHub repository // the following are received from GitHub.com NrOfOpenBugs int // the number of open bugs in that repo NrOfOpenIssues int // number of open issues // the following are received from plausible.io Visitors7d string Pageviews7d string Visitors30d string Pageviews30d string Visitors12m string Pageviews12m string } Each of the three time periods (7D, 30d, 12m) need a distinct call to plausible.io, so these can be called in parallel goroutines. See ADR-0004 how Goroutines are used. Consequences The various funcs calling Plausible and GitHub need to return parts of these structs, so we can collect "
},

{
    "id": 4,
    "uri": "adrs/0007-migrate-to-single-repo.html",
    "menu": "null",
    "title": "null",
    "text": " 7. migrate-to-single-repo Date: 2023-12-09 Status Accepted Context Previously, the code for status.arc42.org was split into TWO different GitHub repositories: https://github.com/arc42/status.arc42.org-site https://github.com/arc42/site-usage-statistics That resulted from history: Between January and October 2023, only the static (Jekyll-based) site was available. Only then the (dynamic, Golang-based) app was developed. This two-repo situation suffers from some drawbacks: It is unclear: * where to put documentation? * where to open/maintain bugs, issues and/or feature-requests? Updating or releasing often needs coordinated changes in both repositories. Decision Migrate both repositories, move content into the status-repo. Consequences build processes have to be updated See GitHub issue #72 "
},

{
    "id": 5,
    "uri": "adrs/0005-concurrent-access-to-external-apis.html",
    "menu": "null",
    "title": "null",
    "text": " 5. concurrent access to external APIs Date: 2023-12-04 Status Accepted Context For the 7+ arc42 websites we need 3 queries each to plausible.io (for the three time periods) plus one graphql query to GitHub. Performing these 28 queries sequentially takes approx. 4 seconds on average, which seems too slow for the website. Therefor, we evaluated concurrent access to these APIs. Decision Goroutines are an established way of performing concurrent processing in Golang. Their approach is well-documented and they are fairly easy to implement. But we don't want the additional complexity of channels, but stick to the easiest programming model possible: refactor the funcs calling external APIs to get a distinct pointer to a struct have a sync.WaitGroup in the func surrounding these calls do not use a mutex. Consequences "
},

{
    "id": 6,
    "uri": "adrs/0009-format-large-numbers-with-separator-chars.html",
    "menu": "null",
    "title": "null",
    "text": " 9. format large numbers with separator chars Date: 2023-12-12 Status Accepted Context The (fairly) large access numbers for the sites result in numbers difficult to read (e.g. 805455). Decision Numbers in the generated HTML table shall be formatted using separators, resulting in e.g. 805.455 We decided to use the following way to get separators: p := message.NewPrinter(language.German) myStr := p.Sprintf( &quot;%d&quot;, 1234567) // myStr == &quot;1.234.567&quot; Consequences Some types (e.g. types/TotalsForAllSites) need to carry around both an int PLUS a string representation of the same value. The string is formatted with decimal separators, whereas the numbers aren't. "
},

{
    "id": 7,
    "uri": "adrs/0008-deploy-on-fly-io.html",
    "menu": "null",
    "title": "null",
    "text": " 8. Deploy on fly.io Date: 2023-12-10 Status Accepted Context We want to make the statistic-service available online, so we need to either host a service on-premise or in the cloud. Decision Deploy the (PRODUCTION) service on fly.io , an affordable cloud service provider with a nice developer experience. Consequences some secrets (API-tokens) need to be configured via the fly.io command line tool. for development, the flyctl utility needs to be installed. See their documentation for details. "
},

{
    "id": 8,
    "uri": "adrs/002-use-zerolog-for-logging.html",
    "menu": "null",
    "title": "null",
    "text": " 2. use zerolog for logging Date: 2023-11-27 Status Accepted Context Previously several fmt.Print* function calls were scattered around the code. Especially when run in the fly.io cloud, these normal print statements didn't work properly. Therefore, issue #58 was created to track this problem. Decision We will use zerolog for logging. Consequences a global logger ('log') is made available zerolog has to be imported by all packages "
},

{
    "id": 9,
    "uri": "arc42/arc42.html",
    "menu": "-",
    "title": "image:arc42-logo.png[arc42] Template",
    "text": " Table of Contents Template 1. Introduction and Goals 1.1. Requirements Overview 1.2. Quality Goals 1.3. Stakeholders 2. Architecture Constraints 3. System Scope and Context 3.1. Business Context 3.2. Technical Context 4. Solution Strategy 5. Building Block View 5.1. Whitebox Overall System 5.2. Level 2 5.3. Level 3 6. Runtime View 6.1. &lt;Runtime Scenario 1&gt; 6.2. &lt;Runtime Scenario 2&gt; 6.3. &#8230;&#8203; 6.4. &lt;Runtime Scenario n&gt; 7. Deployment View 7.1. Infrastructure Level 1 7.2. Infrastructure Level 2 8. Cross-cutting Concepts 8.1. &lt;Concept 1&gt; 8.2. &lt;Concept 2&gt; 8.3. &lt;Concept n&gt; 9. Architecture Decisions 10. Quality Requirements 10.1. Quality Tree 10.2. Quality Scenarios 11. Risks and Technical Debts 12. Glossary Template .arc42help {font-size:small; width: 14px; height: 16px; overflow: hidden; position: absolute; right: 0; padding: 2px 0 3px 2px;} .arc42help::before {content: \"?\";} .arc42help:hover {width:auto; height: auto; z-index: 100; padding: 10px;} .arc42help:hover::before {content: \"\";} @media print { .arc42help {display:none;} } About arc42 arc42, the template for documentation of software and system architecture. Template Version 8.2 EN. (based upon AsciiDoc version), January 2023 Created, maintained and &#169; by Dr. Peter Hruschka, Dr. Gernot Starke and contributors. See https://arc42.org . 1. Introduction and Goals 1.1. Requirements Overview 1.2. Quality Goals 1.3. Stakeholders Role/Name Contact Expectations &lt;Role-1&gt; &lt;Contact-1&gt; &lt;Expectation-1&gt; &lt;Role-2&gt; &lt;Contact-2&gt; &lt;Expectation-2&gt; 2. Architecture Constraints 3. System Scope and Context 3.1. Business Context &lt;Diagram or Table&gt; &lt;optionally: Explanation of external domain interfaces&gt; 3.2. Technical Context &lt;Diagram or Table&gt; &lt;optionally: Explanation of technical interfaces&gt; &lt;Mapping Input/Output to Channels&gt; 4. Solution Strategy 5. Building Block View 5.1. Whitebox Overall System &lt;Overview Diagram&gt; Motivation &lt;text explanation&gt; Contained Building Blocks &lt;Description of contained building block (black boxes)&gt; Important Interfaces &lt;Description of important interfaces&gt; 5.1.1. &lt;Name black box 1&gt; &lt;Purpose/Responsibility&gt; &lt;Interface(s)&gt; &lt;(Optional) Quality/Performance Characteristics&gt; &lt;(Optional) Directory/File Location&gt; &lt;(Optional) Fulfilled Requirements&gt; &lt;(optional) Open Issues/Problems/Risks&gt; 5.1.2. &lt;Name black box 2&gt; &lt;black box template&gt; 5.1.3. &lt;Name black box n&gt; &lt;black box template&gt; 5.1.4. &lt;Name interface 1&gt; &#8230;&#8203; 5.1.5. &lt;Name interface m&gt; 5.2. Level 2 5.2.1. White Box &lt;building block 1&gt; &lt;white box template&gt; 5.2.2. White Box &lt;building block 2&gt; &lt;white box template&gt; &#8230;&#8203; 5.2.3. White Box &lt;building block m&gt; &lt;white box template&gt; 5.3. Level 3 5.3.1. White Box &lt;_building block x.1_&gt; &lt;white box template&gt; 5.3.2. White Box &lt;_building block x.2_&gt; &lt;white box template&gt; 5.3.3. White Box &lt;_building block y.1_&gt; &lt;white box template&gt; 6. Runtime View 6.1. &lt;Runtime Scenario 1&gt; &lt;insert runtime diagram or textual description of the scenario&gt; &lt;insert description of the notable aspects of the interactions between the building block instances depicted in this diagram.&gt; 6.2. &lt;Runtime Scenario 2&gt; 6.3. &#8230;&#8203; 6.4. &lt;Runtime Scenario n&gt; 7. Deployment View 7.1. Infrastructure Level 1 &lt;Overview Diagram&gt; Motivation &lt;explanation in text form&gt; Quality and/or Performance Features &lt;explanation in text form&gt; Mapping of Building Blocks to Infrastructure &lt;description of the mapping&gt; 7.2. Infrastructure Level 2 7.2.1. &lt;Infrastructure Element 1&gt; &lt;diagram + explanation&gt; 7.2.2. &lt;Infrastructure Element 2&gt; &lt;diagram + explanation&gt; &#8230;&#8203; 7.2.3. &lt;Infrastructure Element n&gt; &lt;diagram + explanation&gt; 8. Cross-cutting Concepts 8.1. &lt;Concept 1&gt; &lt;explanation&gt; 8.2. &lt;Concept 2&gt; &lt;explanation&gt; &#8230;&#8203; 8.3. &lt;Concept n&gt; &lt;explanation&gt; 9. Architecture Decisions 10. Quality Requirements 10.1. Quality Tree 10.2. Quality Scenarios 11. Risks and Technical Debts 12. Glossary Term Definition &lt;Term-1&gt; &lt;definition-1&gt; &lt;Term-2&gt; &lt;definition-2&gt; "
},

{
    "id": 10,
    "uri": "arc42/chapters/04_solution_strategy.html",
    "menu": "arc42",
    "title": "Solution Strategy",
    "text": " Table of Contents Solution Strategy Solution Strategy "
},

{
    "id": 11,
    "uri": "arc42/chapters/07_deployment_view.html",
    "menu": "arc42",
    "title": "Deployment View",
    "text": " Table of Contents Deployment View Infrastructure Level 1 Infrastructure Level 2 Deployment View Infrastructure Level 1 &lt;Overview Diagram&gt; Motivation &lt;explanation in text form&gt; Quality and/or Performance Features &lt;explanation in text form&gt; Mapping of Building Blocks to Infrastructure &lt;description of the mapping&gt; Infrastructure Level 2 &lt;Infrastructure Element 1&gt; &lt;diagram + explanation&gt; &lt;Infrastructure Element 2&gt; &lt;diagram + explanation&gt; &#8230;&#8203; &lt;Infrastructure Element n&gt; &lt;diagram + explanation&gt; "
},

{
    "id": 12,
    "uri": "arc42/chapters/09_architecture_decisions.html",
    "menu": "arc42",
    "title": "Architecture Decisions",
    "text": " Table of Contents Architecture Decisions Architecture Decisions "
},

{
    "id": 13,
    "uri": "arc42/chapters/05_building_block_view.html",
    "menu": "arc42",
    "title": "Building Block View",
    "text": " Table of Contents Building Block View Whitebox Overall System Level 2 Level 3 Building Block View Whitebox Overall System &lt;Overview Diagram&gt; Motivation &lt;text explanation&gt; Contained Building Blocks &lt;Description of contained building block (black boxes)&gt; Important Interfaces &lt;Description of important interfaces&gt; &lt;Name black box 1&gt; &lt;Purpose/Responsibility&gt; &lt;Interface(s)&gt; &lt;(Optional) Quality/Performance Characteristics&gt; &lt;(Optional) Directory/File Location&gt; &lt;(Optional) Fulfilled Requirements&gt; &lt;(optional) Open Issues/Problems/Risks&gt; &lt;Name black box 2&gt; &lt;black box template&gt; &lt;Name black box n&gt; &lt;black box template&gt; &lt;Name interface 1&gt; &#8230;&#8203; &lt;Name interface m&gt; Level 2 White Box &lt;building block 1&gt; &lt;white box template&gt; White Box &lt;building block 2&gt; &lt;white box template&gt; &#8230;&#8203; White Box &lt;building block m&gt; &lt;white box template&gt; Level 3 White Box &lt;_building block x.1_&gt; &lt;white box template&gt; White Box &lt;_building block x.2_&gt; &lt;white box template&gt; White Box &lt;_building block y.1_&gt; &lt;white box template&gt; "
},

{
    "id": 14,
    "uri": "arc42/chapters/08_concepts.html",
    "menu": "arc42",
    "title": "Cross-cutting Concepts",
    "text": " Table of Contents Cross-cutting Concepts &lt;Concept 1&gt; &lt;Concept 2&gt; &lt;Concept n&gt; Cross-cutting Concepts &lt;Concept 1&gt; &lt;explanation&gt; &lt;Concept 2&gt; &lt;explanation&gt; &#8230;&#8203; &lt;Concept n&gt; &lt;explanation&gt; "
},

{
    "id": 15,
    "uri": "arc42/chapters/10_quality_requirements.html",
    "menu": "arc42",
    "title": "Quality Requirements",
    "text": " Table of Contents Quality Requirements Quality Tree Quality Scenarios Quality Requirements Quality Tree Quality Scenarios "
},

{
    "id": 16,
    "uri": "arc42/chapters/12_glossary.html",
    "menu": "arc42",
    "title": "Glossary",
    "text": " Table of Contents Glossary Glossary Term Definition &lt;Term-1&gt; &lt;definition-1&gt; &lt;Term-2&gt; &lt;definition-2&gt; "
},

{
    "id": 17,
    "uri": "arc42/chapters/11_technical_risks.html",
    "menu": "arc42",
    "title": "Risks and Technical Debts",
    "text": " Table of Contents Risks and Technical Debts Risks and Technical Debts "
},

{
    "id": 18,
    "uri": "arc42/chapters/06_runtime_view.html",
    "menu": "arc42",
    "title": "Runtime View",
    "text": " Table of Contents Runtime View &lt;Runtime Scenario 1&gt; &lt;Runtime Scenario 2&gt; &#8230;&#8203; &lt;Runtime Scenario n&gt; Runtime View &lt;Runtime Scenario 1&gt; &lt;insert runtime diagram or textual description of the scenario&gt; &lt;insert description of the notable aspects of the interactions between the building block instances depicted in this diagram.&gt; &lt;Runtime Scenario 2&gt; &#8230;&#8203; &lt;Runtime Scenario n&gt; "
},

{
    "id": 19,
    "uri": "arc42/chapters/01_introduction_and_goals.html",
    "menu": "arc42",
    "title": "Introduction and Goals",
    "text": " Table of Contents Introduction and Goals Requirements Overview Quality Goals Stakeholders Introduction and Goals Requirements Overview Quality Goals Stakeholders Role/Name Contact Expectations &lt;Role-1&gt; &lt;Contact-1&gt; &lt;Expectation-1&gt; &lt;Role-2&gt; &lt;Contact-2&gt; &lt;Expectation-2&gt; "
},

{
    "id": 20,
    "uri": "arc42/chapters/03_system_scope_and_context.html",
    "menu": "arc42",
    "title": "System Scope and Context",
    "text": " Table of Contents System Scope and Context Business Context Technical Context System Scope and Context Business Context &lt;Diagram or Table&gt; &lt;optionally: Explanation of external domain interfaces&gt; Technical Context &lt;Diagram or Table&gt; &lt;optionally: Explanation of technical interfaces&gt; &lt;Mapping Input/Output to Channels&gt; "
},

{
    "id": 21,
    "uri": "arc42/chapters/02_architecture_constraints.html",
    "menu": "arc42",
    "title": "Architecture Constraints",
    "text": " Table of Contents Architecture Constraints Architecture Constraints "
},

{
    "id": 22,
    "uri": "search.html",
    "menu": "-",
    "title": "search",
    "text": " Search Results "
},

{
    "id": 23,
    "uri": "lunrjsindex.html",
    "menu": "-",
    "title": "null",
    "text": " will be replaced by the index "
},

];
