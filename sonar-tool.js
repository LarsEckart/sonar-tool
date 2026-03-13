#!/usr/bin/env bun

/**
 * Sonar Tool CLI
 * 
 * Search and retrieve issues from SonarCloud.
 * 
 * Setup:
 *   export SONAR_TOKEN="your_token_here"
 *   export SONAR_ORG="your_organization"  # optional default
 * 
 * Usage:
 *   sonar-tool.js --project my_project
 *   sonar-tool.js --project my_project --types BUG,VULNERABILITY
 *   sonar-tool.js --project my_project --severities CRITICAL,BLOCKER
 *   sonar-tool.js --org my-org  # all issues in organization
 *   sonar-tool.js --project my_project --branch feature/xyz
 *   sonar-tool.js --project my_project --new  # only new issues (since leak period)
 *   sonar-tool.js --project my_project -n 20  # limit results
 *   sonar-tool.js --project my_project --json  # JSON output
 */

const https = require('https');
const { URL } = require('url');

const BASE_URL = 'https://sonarcloud.io';

function getToken() {
  const token = process.env.SONAR_TOOL_TOKEN || process.env.SONAR_TOKEN;
  if (!token) {
    console.error('Error: SONAR_TOOL_TOKEN or SONAR_TOKEN environment variable not set');
    console.error('Generate a token at: https://sonarcloud.io/account/security');
    process.exit(1);
  }
  return token;
}

function parseArgs(args) {
  const opts = {
    listProjects: false,
    project: null,
    org: process.env.SONAR_ORG || null,
    branch: null,
    pullRequest: null,
    types: null,           // CODE_SMELL, BUG, VULNERABILITY
    severities: null,      // INFO, MINOR, MAJOR, CRITICAL, BLOCKER
    impactSeverities: null, // INFO, LOW, MEDIUM, HIGH, BLOCKER
    impactQualities: null,  // MAINTAINABILITY, RELIABILITY, SECURITY
    statuses: null,        // OPEN, CONFIRMED, FALSE_POSITIVE, ACCEPTED, FIXED
    tags: null,
    rules: null,
    assignee: null,
    author: null,
    languages: null,
    createdAfter: null,
    createdBefore: null,
    createdInLast: null,
    sinceLeakPeriod: false,
    resolved: null,
    limit: 100,
    page: 1,
    sort: null,            // SEVERITY, CREATION_DATE, UPDATE_DATE, etc.
    asc: null,
    json: false,
    markdown: false,
    full: false,
    help: false,
  };

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    const next = args[i + 1];

    switch (arg) {
      case '-h':
      case '--help':
        opts.help = true;
        break;
      case '-p':
      case '--project':
        // If next arg is missing or is another flag, list projects
        if (!next || next.startsWith('-')) {
          opts.listProjects = true;
        } else {
          opts.project = next;
          i++;
        }
        break;
      case '-o':
      case '--org':
      case '--organization':
        opts.org = next;
        i++;
        break;
      case '-b':
      case '--branch':
        opts.branch = next;
        i++;
        break;
      case '--pr':
      case '--pull-request':
        opts.pullRequest = next;
        i++;
        break;
      case '-t':
      case '--types':
        opts.types = next;
        i++;
        break;
      case '-s':
      case '--severities':
        opts.severities = next;
        i++;
        break;
      case '--impact-severities':
        opts.impactSeverities = next;
        i++;
        break;
      case '--impact-qualities':
      case '--qualities':
        opts.impactQualities = next;
        i++;
        break;
      case '--statuses':
        opts.statuses = next;
        i++;
        break;
      case '--tags':
        opts.tags = next;
        i++;
        break;
      case '--rules':
        opts.rules = next;
        i++;
        break;
      case '--assignee':
        opts.assignee = next;
        i++;
        break;
      case '--author':
        opts.author = next;
        i++;
        break;
      case '-l':
      case '--languages':
        opts.languages = next;
        i++;
        break;
      case '--created-after':
        opts.createdAfter = next;
        i++;
        break;
      case '--created-before':
        opts.createdBefore = next;
        i++;
        break;
      case '--created-in-last':
        opts.createdInLast = next;
        i++;
        break;
      case '--new':
      case '--since-leak-period':
        opts.sinceLeakPeriod = true;
        break;
      case '--resolved':
        opts.resolved = next === 'true' || next === 'yes';
        i++;
        break;
      case '--unresolved':
        opts.resolved = false;
        break;
      case '-n':
      case '--limit':
        opts.limit = parseInt(next, 10);
        i++;
        break;
      case '--page':
        opts.page = parseInt(next, 10);
        i++;
        break;
      case '--sort':
        opts.sort = next;
        i++;
        break;
      case '--asc':
        opts.asc = true;
        break;
      case '--desc':
        opts.asc = false;
        break;
      case '--json':
        opts.json = true;
        break;
      case '--md':
      case '--markdown':
        opts.markdown = true;
        break;
      case '--full':
        opts.full = true;
        break;
    }
  }

  return opts;
}

function printHelp() {
  console.log(`Sonar Tool CLI

Usage: sonar-tool.js [options]

Required:
  -p, --project <key>       Project key (or use -p alone to list all projects)
  -o, --org <key>           Organization key (or set SONAR_ORG env var)

Filters:
  -b, --branch <name>       Branch name
  --pr, --pull-request <id> Pull request ID
  -t, --types <list>        Issue types: CODE_SMELL, BUG, VULNERABILITY
  -s, --severities <list>   Severities: INFO, MINOR, MAJOR, CRITICAL, BLOCKER
  --impact-severities <list> Impact severities: INFO, LOW, MEDIUM, HIGH, BLOCKER
  --qualities <list>        Software qualities: MAINTAINABILITY, RELIABILITY, SECURITY
  --statuses <list>         Statuses: OPEN, CONFIRMED, FALSE_POSITIVE, ACCEPTED, FIXED
  --tags <list>             Comma-separated tags
  --rules <list>            Rule keys (e.g., java:S1234)
  --assignee <login>        Assignee login (use __me__ for current user)
  --author <email>          SCM author
  -l, --languages <list>    Languages (e.g., java,js,py)

Time filters:
  --created-after <date>    Issues created after date (YYYY-MM-DD)
  --created-before <date>   Issues created before date
  --created-in-last <span>  Created in last period (e.g., 1m2w = 1 month 2 weeks)
  --new, --since-leak-period Only new issues since leak period

Resolution:
  --resolved true/false     Filter by resolution status
  --unresolved              Shorthand for --resolved false

Output:
  -n, --limit <num>         Max results (default: 100, max: 500)
  --page <num>              Page number (default: 1)
  --sort <field>            Sort by: SEVERITY, CREATION_DATE, UPDATE_DATE, etc.
  --asc / --desc            Sort direction
  --json                    Output raw JSON
  --md, --markdown          Output as markdown
  --full                    Show all issue details (default: concise)

Environment:
  SONAR_TOKEN               API token (required)
  SONAR_ORG                 Default organization

Examples:
  sonar-tool.js -o my-org -p                # List all projects
  sonar-tool.js -o my-org -p --markdown     # List projects as markdown
  sonar-tool.js -p my_project -o my-org     # Issues for a project
  sonar-tool.js -p my_project --types BUG,VULNERABILITY
  sonar-tool.js -p my_project --severities CRITICAL,BLOCKER --unresolved
  sonar-tool.js -p my_project --new -n 50
  sonar-tool.js -p my_project --branch main --created-in-last 7d
  sonar-tool.js -o my-org --qualities SECURITY --json
`);
}

function buildSearchParams(opts) {
  const params = new URLSearchParams();

  if (opts.project) {
    params.set('componentKeys', opts.project);
  }
  if (opts.org) {
    params.set('organization', opts.org);
  }
  if (opts.branch) {
    params.set('branch', opts.branch);
  }
  if (opts.pullRequest) {
    params.set('pullRequest', opts.pullRequest);
  }
  if (opts.types) {
    params.set('types', opts.types);
  }
  if (opts.severities) {
    params.set('severities', opts.severities);
  }
  if (opts.impactSeverities) {
    params.set('impactSeverities', opts.impactSeverities);
  }
  if (opts.impactQualities) {
    params.set('impactSoftwareQualities', opts.impactQualities);
  }
  if (opts.statuses) {
    params.set('issueStatuses', opts.statuses);
  }
  if (opts.tags) {
    params.set('tags', opts.tags);
  }
  if (opts.rules) {
    params.set('rules', opts.rules);
  }
  if (opts.assignee) {
    params.set('assignees', opts.assignee);
  }
  if (opts.author) {
    params.set('author', opts.author);
  }
  if (opts.languages) {
    params.set('languages', opts.languages);
  }
  if (opts.createdAfter) {
    params.set('createdAfter', opts.createdAfter);
  }
  if (opts.createdBefore) {
    params.set('createdBefore', opts.createdBefore);
  }
  if (opts.createdInLast) {
    params.set('createdInLast', opts.createdInLast);
  }
  if (opts.sinceLeakPeriod) {
    params.set('sinceLeakPeriod', 'true');
  }
  if (opts.resolved !== null) {
    params.set('resolved', opts.resolved ? 'true' : 'false');
  }
  if (opts.sort) {
    params.set('s', opts.sort);
  }
  if (opts.asc !== null) {
    params.set('asc', opts.asc ? 'true' : 'false');
  }

  params.set('ps', Math.min(opts.limit, 500).toString());
  params.set('p', opts.page.toString());

  // Request additional fields for richer output
  params.set('additionalFields', 'rules');

  return params;
}

function buildProjectsParams(opts) {
  const params = new URLSearchParams();
  
  if (opts.org) {
    params.set('organization', opts.org);
  }
  
  params.set('ps', Math.min(opts.limit, 500).toString());
  params.set('p', opts.page.toString());
  
  return params;
}

function formatProject(project) {
  return project.key;
}

function formatProjectMarkdown(project) {
  return `- \`${project.key}\``;
}

function formatProjectsOutput(data, opts) {
  if (opts.json) {
    return JSON.stringify(data, null, 2);
  }

  const { components: projects, paging } = data;

  if (opts.markdown) {
    const lines = [];
    lines.push('## Projects');
    lines.push('');
    projects.forEach((project) => {
      lines.push(formatProjectMarkdown(project));
    });
    if (paging.total > paging.pageIndex * paging.pageSize) {
      const remaining = paging.total - paging.pageIndex * paging.pageSize;
      lines.push('');
      lines.push(`*${remaining} more projects. Use \`--page ${paging.pageIndex + 1}\` for next page.*`);
    }
    return lines.join('\n');
  }

  // Plain text - just list the keys
  const lines = [];
  projects.forEach((project) => {
    lines.push(formatProject(project));
  });
  if (paging.total > paging.pageIndex * paging.pageSize) {
    const remaining = paging.total - paging.pageIndex * paging.pageSize;
    lines.push(`# ... ${remaining} more projects. Use --page ${paging.pageIndex + 1} to see next page.`);
  }
  return lines.join('\n');
}

function request(path, token) {
  return new Promise((resolve, reject) => {
    const url = new URL(path, BASE_URL);

    const options = {
      hostname: url.hostname,
      path: url.pathname + url.search,
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Accept': 'application/json',
      },
    };

    const req = https.request(options, (res) => {
      let data = '';
      res.on('data', (chunk) => data += chunk);
      res.on('end', () => {
        if (res.statusCode >= 400) {
          reject(new Error(`HTTP ${res.statusCode}: ${data}`));
        } else {
          try {
            resolve(JSON.parse(data));
          } catch (e) {
            reject(new Error(`Invalid JSON response: ${data}`));
          }
        }
      });
    });

    req.on('error', reject);
    req.end();
  });
}

function formatIssueConcise(issue, rules, index) {
  const rule = rules.find(r => r.key === issue.rule) || {};
  const lines = [];

  // File location
  const component = issue.component || '';
  const file = component.includes(':') ? component.split(':').slice(1).join(':') : component;
  const location = issue.line ? `${file}:${issue.line}` : file;

  lines.push(`--- Issue ${index} ---`);
  lines.push(`Rule: ${rule.name || issue.rule}`);
  lines.push(`Message: ${issue.message}`);
  lines.push(`File: ${location}`);
  lines.push('');

  return lines.join('\n');
}

function formatIssueFull(issue, rules, index) {
  const rule = rules.find(r => r.key === issue.rule) || {};
  const lines = [];

  lines.push(`--- Issue ${index} ---`);
  lines.push(`Rule: ${issue.rule}`);
  if (rule.name) {
    lines.push(`Rule Name: ${rule.name}`);
  }
  lines.push(`Message: ${issue.message}`);
  lines.push(`Severity: ${issue.severity || 'N/A'}`);
  
  if (issue.impacts && issue.impacts.length > 0) {
    const impacts = issue.impacts.map(i => `${i.softwareQuality}:${i.severity}`).join(', ');
    lines.push(`Impacts: ${impacts}`);
  }
  
  lines.push(`Type: ${issue.type || 'N/A'}`);
  lines.push(`Status: ${issue.issueStatus || issue.status}`);
  
  if (issue.cleanCodeAttribute) {
    lines.push(`Clean Code: ${issue.cleanCodeAttributeCategory} / ${issue.cleanCodeAttribute}`);
  }

  // File location
  const component = issue.component || '';
  const file = component.includes(':') ? component.split(':').slice(1).join(':') : component;
  if (file) {
    const location = issue.line ? `${file}:${issue.line}` : file;
    lines.push(`File: ${location}`);
  }

  if (issue.effort) {
    lines.push(`Effort: ${issue.effort}`);
  }

  if (issue.tags && issue.tags.length > 0) {
    lines.push(`Tags: ${issue.tags.join(', ')}`);
  }

  if (issue.assignee) {
    lines.push(`Assignee: ${issue.assignee}`);
  }

  if (issue.author) {
    lines.push(`Author: ${issue.author}`);
  }

  lines.push(`Created: ${issue.creationDate}`);
  if (issue.updateDate && issue.updateDate !== issue.creationDate) {
    lines.push(`Updated: ${issue.updateDate}`);
  }

  lines.push(`Key: ${issue.key}`);
  lines.push('');

  return lines.join('\n');
}

function severityEmoji(severity) {
  const map = {
    'BLOCKER': '🔴',
    'CRITICAL': '🟠',
    'MAJOR': '🟡',
    'MINOR': '🟢',
    'INFO': '⚪',
    'HIGH': '🔴',
    'MEDIUM': '🟡',
    'LOW': '🟢',
  };
  return map[severity] || '⚪';
}

function typeEmoji(type) {
  const map = {
    'BUG': '🐛',
    'VULNERABILITY': '🔓',
    'CODE_SMELL': '👃',
    'SECURITY_HOTSPOT': '🔥',
  };
  return map[type] || '📝';
}

function formatIssueMarkdown(issue, rules, index) {
  const rule = rules.find(r => r.key === issue.rule) || {};
  const lines = [];

  const severity = issue.severity || 'N/A';
  const type = issue.type || 'N/A';
  
  // Header with severity and type icons
  lines.push(`### ${index}. ${severityEmoji(severity)} ${typeEmoji(type)} ${issue.message}`);
  lines.push('');

  // File location as code block style
  const component = issue.component || '';
  const file = component.includes(':') ? component.split(':').slice(1).join(':') : component;
  if (file) {
    const location = issue.line ? `${file}:${issue.line}` : file;
    lines.push(`📁 \`${location}\``);
    lines.push('');
  }

  // Details table
  lines.push('| Property | Value |');
  lines.push('|----------|-------|');
  lines.push(`| **Rule** | \`${issue.rule}\` ${rule.name ? `- ${rule.name}` : ''} |`);
  lines.push(`| **Severity** | ${severityEmoji(severity)} ${severity} |`);
  lines.push(`| **Type** | ${typeEmoji(type)} ${type} |`);
  lines.push(`| **Status** | ${issue.issueStatus || issue.status} |`);

  if (issue.impacts && issue.impacts.length > 0) {
    const impacts = issue.impacts.map(i => `${i.softwareQuality}: ${severityEmoji(i.severity)} ${i.severity}`).join(', ');
    lines.push(`| **Impacts** | ${impacts} |`);
  }

  if (issue.cleanCodeAttribute) {
    lines.push(`| **Clean Code** | ${issue.cleanCodeAttributeCategory} / ${issue.cleanCodeAttribute} |`);
  }

  if (issue.effort) {
    lines.push(`| **Effort** | ${issue.effort} |`);
  }

  if (issue.tags && issue.tags.length > 0) {
    const tags = issue.tags.map(t => `\`${t}\``).join(' ');
    lines.push(`| **Tags** | ${tags} |`);
  }

  if (issue.assignee) {
    lines.push(`| **Assignee** | ${issue.assignee} |`);
  }

  if (issue.author) {
    lines.push(`| **Author** | ${issue.author} |`);
  }

  lines.push(`| **Created** | ${issue.creationDate} |`);
  
  if (issue.updateDate && issue.updateDate !== issue.creationDate) {
    lines.push(`| **Updated** | ${issue.updateDate} |`);
  }

  lines.push(`| **Key** | \`${issue.key}\` |`);
  lines.push('');

  return lines.join('\n');
}

function formatMarkdown(data, opts) {
  const { issues, rules = [], paging } = data;
  const lines = [];

  // Header
  lines.push('# SonarCloud Issues Report');
  lines.push('');
  lines.push(`> Found **${paging.total}** issues (showing ${issues.length} on page ${paging.pageIndex})`);
  lines.push('');

  // Summary by severity if we have issues
  if (issues.length > 0) {
    const bySeverity = {};
    const byType = {};
    issues.forEach(issue => {
      const sev = issue.severity || 'N/A';
      const type = issue.type || 'N/A';
      bySeverity[sev] = (bySeverity[sev] || 0) + 1;
      byType[type] = (byType[type] || 0) + 1;
    });

    lines.push('## Summary (this page)');
    lines.push('');
    
    const sevOrder = ['BLOCKER', 'CRITICAL', 'MAJOR', 'MINOR', 'INFO'];
    const sevSummary = sevOrder
      .filter(s => bySeverity[s])
      .map(s => `${severityEmoji(s)} ${s}: ${bySeverity[s]}`)
      .join(' | ');
    if (sevSummary) {
      lines.push(`**By Severity:** ${sevSummary}`);
      lines.push('');
    }

    const typeSummary = Object.entries(byType)
      .map(([t, count]) => `${typeEmoji(t)} ${t}: ${count}`)
      .join(' | ');
    if (typeSummary) {
      lines.push(`**By Type:** ${typeSummary}`);
      lines.push('');
    }

    lines.push('---');
    lines.push('');
  }

  // Issues
  lines.push('## Issues');
  lines.push('');

  issues.forEach((issue, i) => {
    lines.push(formatIssueMarkdown(issue, rules, i + 1 + (paging.pageIndex - 1) * paging.pageSize));
  });

  // Pagination note
  if (paging.total > paging.pageIndex * paging.pageSize) {
    const remaining = paging.total - paging.pageIndex * paging.pageSize;
    lines.push('---');
    lines.push('');
    lines.push(`*${remaining} more issues available. Use \`--page ${paging.pageIndex + 1}\` to see next page.*`);
  }

  return lines.join('\n');
}

function formatOutput(data, opts) {
  if (opts.json) {
    return JSON.stringify(data, null, 2);
  }

  if (opts.markdown) {
    return formatMarkdown(data, opts);
  }

  const { issues, rules = [], paging } = data;
  const lines = [];

  lines.push(`Found ${paging.total} issues (showing ${issues.length} on page ${paging.pageIndex})`);
  lines.push('');

  const formatter = opts.full ? formatIssueFull : formatIssueConcise;
  issues.forEach((issue, i) => {
    lines.push(formatter(issue, rules, i + 1 + (paging.pageIndex - 1) * paging.pageSize));
  });

  if (paging.total > paging.pageIndex * paging.pageSize) {
    const remaining = paging.total - paging.pageIndex * paging.pageSize;
    lines.push(`... ${remaining} more issues. Use --page ${paging.pageIndex + 1} to see next page.`);
  }

  return lines.join('\n');
}

async function validateOrg(org, token) {
  const path = `/api/organizations/search?organizations=${encodeURIComponent(org)}`;
  const data = await request(path, token);
  if (!data.organizations || data.organizations.length === 0) {
    throw new Error(`Organization '${org}' not found`);
  }
}

async function main() {
  const args = process.argv.slice(2);
  const opts = parseArgs(args);

  if (opts.help) {
    printHelp();
    process.exit(0);
  }

  const token = getToken();

  // Validate org if provided
  if (opts.org) {
    try {
      await validateOrg(opts.org, token);
    } catch (err) {
      console.error(`Error: ${err.message}`);
      process.exit(1);
    }
  }

  // List projects mode
  if (opts.listProjects) {
    if (!opts.org) {
      console.error('Error: --org is required to list projects');
      console.error('Use --help for usage information');
      process.exit(1);
    }
    const params = buildProjectsParams(opts);
    const path = `/api/components/search_projects?${params.toString()}`;

    try {
      const data = await request(path, token);
      console.log(formatProjectsOutput(data, opts));
    } catch (err) {
      console.error(`Error: ${err.message}`);
      process.exit(1);
    }
    return;
  }

  // Issues search mode
  if (!opts.project && !opts.org) {
    console.error('Error: Either --project or --org is required');
    console.error('Use --help for usage information');
    process.exit(1);
  }

  const params = buildSearchParams(opts);
  const path = `/api/issues/search?${params.toString()}`;

  try {
    const data = await request(path, token);
    console.log(formatOutput(data, opts));
  } catch (err) {
    console.error(`Error: ${err.message}`);
    process.exit(1);
  }
}

main();
