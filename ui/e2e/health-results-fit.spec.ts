import { expect, type Locator, type Page, test } from '@playwright/test';
import { skipSetupWizard } from './helpers/auth';

/**
 * Health check results must fit their card and list failures first (#461).
 *
 * The results are mocked rather than run: a test host reaches none of these
 * services, so every row would fail the same way and neither the widths nor
 * the ordering would be exercised. The values are the widest realistic ones
 * -- long check names, FQDN hosts and full URLs -- and in every section the
 * failing row is configured LAST, which is where an operator's newest (and
 * most likely broken) check lands.
 */

const LONG_HOST = 'pacs-archive-primary.radiology.east-campus.example-hospital.org';

const ok = { success: true } as const;
const failed = { success: false, error: `dial tcp ${LONG_HOST}:443: i/o timeout` } as const;

const RESULTS = {
  hasTests: true,
  pingResults: [
    { name: 'core-router-1.example.org', latency: 1.4, packetLoss: 0, jitter: 0.2, ...ok },
    {
      name: `Radiology PACS archive (${LONG_HOST})`,
      latency: 0,
      packetLoss: 100,
      ...failed,
    },
  ],
  tcpResults: [
    { name: 'erp.example.org:443', latency: 12, ...ok },
    { name: `${LONG_HOST}:11112`, latency: 0, ...failed },
  ],
  udpResults: [
    { name: 'ntp.example.org:123', latency: 3, ...ok },
    { name: `${LONG_HOST}:514`, latency: 0, ...failed },
  ],
  httpResults: [
    {
      name: 'https://intranet.example.org/status',
      latency: 180,
      status: 200,
      dnsLatency: 10,
      tcpConnect: 20,
      tlsLatency: 40,
      ttfbLatency: 90,
      tlsVersion: 'TLS 1.3',
      certDaysLeft: 200,
      certStatus: 'success',
      certExpiry: '2027-04-18',
      certIssuer: 'DigiCert Global G2 TLS RSA SHA256 2020 CA1',
      ...ok,
    },
    { name: `https://${LONG_HOST}/wado-rs/studies`, latency: 0, ...failed },
  ],
  enterpriseResults: {
    sqlResults: [
      {
        name: 'Billing database',
        driver: 'postgres',
        host: 'billing-db.example.org',
        port: 5432,
        database: 'billing',
        connectTimeMs: 4,
        queryTimeMs: 2,
        totalTimeMs: 6,
        serverVersion: 'PostgreSQL 17.2 on x86_64-pc-linux-gnu, compiled by gcc 14.2.0',
        ...ok,
      },
      {
        name: 'Radiology information system database',
        driver: 'sqlserver',
        host: LONG_HOST,
        port: 1433,
        database: 'ris',
        connectTimeMs: 0,
        totalTimeMs: 5000,
        ...failed,
      },
    ],
    fileShareResults: [
      {
        name: 'Home directories',
        protocol: 'smb',
        host: 'files.example.org',
        share: 'home',
        connectTimeMs: 8,
        readSpeedMbps: 112.4,
        writeSpeedMbps: 98.1,
        totalTimeMs: 40,
        ...ok,
      },
      {
        name: 'Imaging export share',
        protocol: 'nfs',
        host: LONG_HOST,
        share: 'exports/imaging/outbound',
        connectTimeMs: 0,
        totalTimeMs: 5000,
        ...failed,
      },
    ],
    ldapResults: [
      {
        name: 'Directory',
        host: 'dc01.corp.example.org',
        port: 636,
        useTls: true,
        connectTimeMs: 5,
        bindTimeMs: 3,
        totalTimeMs: 8,
        serverInfo: 'Microsoft Active Directory, forest functional level 2016',
        ...ok,
      },
      {
        name: 'Clinical directory',
        host: LONG_HOST,
        port: 636,
        useTls: true,
        connectTimeMs: 0,
        totalTimeMs: 5000,
        ...failed,
      },
    ],
  },
  videoResults: {
    rtspResults: [
      {
        name: 'Lobby camera',
        url: 'rtsp://cam-lobby.example.org:554/stream1',
        connectTimeMs: 30,
        codec: 'H.264',
        resolution: '1920x1080',
        ...ok,
      },
      {
        name: 'Operating theatre 4 camera',
        url: `rtsp://${LONG_HOST}:554/live/theatre-4/main`,
        connectTimeMs: 0,
        ...failed,
      },
    ],
  },
  medicalResults: {
    dicomResults: [
      {
        name: 'Modality worklist',
        host: 'mwl.example.org',
        port: 104,
        aeTitle: 'SEED_SCU',
        connectTimeMs: 3,
        echoTimeMs: 2,
        totalTimeMs: 5,
        serverAeTitle: 'MWL_SCP',
        ...ok,
      },
      {
        name: 'PACS archive',
        host: LONG_HOST,
        port: 11112,
        aeTitle: 'SEED_DIAGNOSTIC_SCU',
        connectTimeMs: 0,
        totalTimeMs: 5000,
        ...failed,
      },
    ],
    hl7Results: [
      {
        name: 'ADT feed',
        host: 'hl7-engine.example.org',
        port: 2575,
        connectTimeMs: 3,
        responseTimeMs: 12,
        totalTimeMs: 15,
        ackCode: 'AA',
        serverVersion: '2.5.1',
        ...ok,
      },
      {
        name: 'Results interface',
        host: LONG_HOST,
        port: 2575,
        connectTimeMs: 0,
        totalTimeMs: 5000,
        ...failed,
      },
    ],
    fhirResults: [
      {
        name: 'Patient API',
        baseUrl: 'https://fhir.example.org/r4',
        connectTimeMs: 20,
        responseTimeMs: 60,
        totalTimeMs: 80,
        fhirVersion: '4.0.1',
        resourceCount: 145,
        serverName: 'HAPI FHIR Server',
        ...ok,
      },
      {
        name: 'Imaging study API',
        baseUrl: `https://${LONG_HOST}/fhir/r4`,
        connectTimeMs: 0,
        totalTimeMs: 5000,
        ...failed,
      },
    ],
  },
  educationResults: {
    ltiResults: [
      {
        name: 'Learning platform',
        launchUrl: 'https://lms.example.edu/lti/launch',
        connectTimeMs: 25,
        totalTimeMs: 90,
        ltiVersion: '1.3',
        ...ok,
      },
      {
        name: 'Clinical training portal',
        launchUrl: `https://${LONG_HOST}/lti/1.3/launch`,
        connectTimeMs: 0,
        totalTimeMs: 5000,
        ...failed,
      },
    ],
  },
  industrialResults: {
    opcuaResults: [
      {
        name: 'Chiller plant',
        endpointUrl: 'opc.tcp://chiller-plc.example.org:4840',
        connectTimeMs: 12,
        browseTimeMs: 8,
        totalTimeMs: 20,
        securityMode: 'SignAndEncrypt',
        serverState: 'Running',
        productName: 'Siemens SIMATIC S7-1500 OPC UA Server',
        ...ok,
      },
      {
        name: 'Medical gas monitoring',
        endpointUrl: `opc.tcp://${LONG_HOST}:4840/freeopcua/server`,
        connectTimeMs: 0,
        totalTimeMs: 5000,
        ...failed,
      },
    ],
    modbusResults: [
      {
        name: 'Generator controller',
        host: 'genset.example.org',
        port: 502,
        unitId: 1,
        connectTimeMs: 4,
        readTimeMs: 2,
        totalTimeMs: 6,
        registerValue: 4660,
        ...ok,
      },
      {
        name: 'Isolation room pressure',
        host: LONG_HOST,
        port: 502,
        unitId: 17,
        connectTimeMs: 0,
        totalTimeMs: 5000,
        ...failed,
      },
    ],
  },
} as const;

/** The names of the failing row and the passing row in each section. */
const SECTIONS: { title: string; failing: string; passing: string }[] = [
  { title: 'Ping', failing: RESULTS.pingResults[1].name, passing: RESULTS.pingResults[0].name },
  { title: 'TCP Ports', failing: RESULTS.tcpResults[1].name, passing: RESULTS.tcpResults[0].name },
  { title: 'UDP Ports', failing: RESULTS.udpResults[1].name, passing: RESULTS.udpResults[0].name },
  // A passing HTTP row carries its status code after the URL.
  {
    title: 'HTTP',
    failing: RESULTS.httpResults[1].name,
    passing: `${RESULTS.httpResults[0].name} (${RESULTS.httpResults[0].status})`,
  },
  {
    title: 'Database',
    failing: 'Radiology information system database',
    passing: 'Billing database',
  },
  { title: 'File Shares', failing: 'Imaging export share', passing: 'Home directories' },
  { title: 'LDAP', failing: 'Clinical directory', passing: 'Directory' },
  { title: 'RTSP Video', failing: 'Operating theatre 4 camera', passing: 'Lobby camera' },
  { title: 'DICOM', failing: 'PACS archive', passing: 'Modality worklist' },
  { title: 'HL7 MLLP', failing: 'Results interface', passing: 'ADT feed' },
  { title: 'FHIR R4', failing: 'Imaging study API', passing: 'Patient API' },
  { title: 'LTI/LMS', failing: 'Clinical training portal', passing: 'Learning platform' },
  { title: 'OPC-UA', failing: 'Medical gas monitoring', passing: 'Chiller plant' },
  { title: 'Modbus TCP', failing: 'Isolation room pressure', passing: 'Generator controller' },
];

async function openHealthCard(page: Page, width: number): Promise<Locator> {
  await page.route('**/api/v1/telemetry/probes/run', (route) => route.fulfill({ json: RESULTS }));
  await page.setViewportSize({ width, height: 900 });
  await skipSetupWizard(page);
  await page.goto('/performance');
  const card = page.getByTestId('card').and(page.getByLabel('Health Checks', { exact: true }));
  await card.getByTestId('health-check-run').click();
  // Every section must be rendered before anything is measured, or the
  // assertions pass against a card that is still loading.
  await expect(card.getByText(SECTIONS.at(-1)?.failing ?? '', { exact: true })).toBeVisible({
    timeout: 10000,
  });
  return card;
}

/**
 * Every visible element inside the card whose box reaches past the card's own
 * box, plus the card itself if it scrolls. A truncated span is fine: its box
 * stays inside and only its text is clipped, with the full value on a tooltip.
 */
async function overflowPastCard(card: Locator): Promise<string[]> {
  return card.evaluate((root) => {
    const edge = root.getBoundingClientRect();
    const out: string[] = [];
    if (root.scrollWidth > root.clientWidth) {
      out.push(`card scrolls by ${root.scrollWidth - root.clientWidth}px`);
    }
    for (const el of Array.from(root.querySelectorAll<HTMLElement>('*'))) {
      const r = el.getBoundingClientRect();
      if (r.width === 0 || r.height === 0) {
        continue;
      }
      const past = Math.max(r.right - edge.right, edge.left - r.left);
      if (past > 1) {
        out.push(`${el.tagName.toLowerCase()} "${el.textContent?.trim().slice(0, 60)}" +${past}px`);
      }
    }
    return out;
  });
}

for (const width of [390, 768, 1280, 1440]) {
  test(`health check results fit their card at ${width}px`, async ({ page }) => {
    const card = await openHealthCard(page, width);

    const past = await overflowPastCard(card);
    expect(past, `health check content reaches past the card at ${width}px`).toEqual([]);

    const pageOverflow = await page.evaluate(
      () => document.documentElement.scrollWidth - window.innerWidth,
    );
    expect(pageOverflow, `the page scrolls sideways at ${width}px`).toBeLessThanOrEqual(0);

    // Fitting must not be bought by breaking short values: a long certificate
    // issuer squeezed "TLS 1.3" onto two lines. One line is shorter than
    // twice its own line height.
    const tls = card.getByText('TLS 1.3', { exact: true });
    const box = await tls.boundingBox();
    const lineHeight = await tls.evaluate((el) =>
      Number.parseFloat(getComputedStyle(el).lineHeight),
    );
    if (!box) {
      throw new Error('the TLS version was not measurable');
    }
    expect(box.height, `the TLS version wrapped at ${width}px`).toBeLessThan(lineHeight * 2);
  });
}

test('each health check section lists its failures first', async ({ page }) => {
  const card = await openHealthCard(page, 1440);

  for (const section of SECTIONS) {
    const failing = await card.getByText(section.failing, { exact: true }).first().boundingBox();
    const passing = await card.getByText(section.passing, { exact: true }).first().boundingBox();
    if (!(failing && passing)) {
      throw new Error(`the ${section.title} rows were not measurable`);
    }
    expect(
      failing.y,
      `${section.title}: the failing check is listed after a passing one`,
    ).toBeLessThan(passing.y);
  }
});
