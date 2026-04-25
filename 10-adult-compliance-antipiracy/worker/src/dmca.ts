export interface DmcaDraftInput {
  toAddress: string;
  fromAddress: string;
  replyTo: string;
  claimantName: string;
  workDescription: string;
  referenceUrl: string;
  candidateUrl: string;
  assetLabel: string;
  matchScore: number;
  detectionMethod: string;
  detectedAt: string;
  signature: string;
}

export function renderDmcaDraft(input: DmcaDraftInput): string {
  return [
    "Operational placeholder only. This draft is not legal advice and must be reviewed before sending.",
    "",
    `To: ${input.toAddress}`,
    `From: ${input.fromAddress}`,
    `Reply-To: ${input.replyTo}`,
    `Subject: Copyright takedown notice for ${input.candidateUrl}`,
    "",
    `I am ${input.claimantName}, the copyright owner or a person authorised to act on behalf of the copyright owner.`,
    "",
    "Copyrighted work:",
    input.workDescription,
    "",
    `Authorised reference location: ${input.referenceUrl}`,
    `Material claimed to be infringing: ${input.candidateUrl}`,
    "",
    "Match details:",
    `- Asset label: ${input.assetLabel}`,
    `- Match score: ${input.matchScore.toFixed(3)}`,
    `- Detection method: ${input.detectionMethod}`,
    `- Detected at: ${input.detectedAt}`,
    "",
    "I have a good-faith belief that the use of the material described above is not authorised by the copyright owner, its agent, or the law.",
    "The information in this notification is accurate. Under penalty of perjury, I swear that I am the copyright owner or am authorised to act on behalf of the owner.",
    "",
    `Electronic signature: ${input.signature}`,
    "",
    "Human review required before send: confirm rights ownership, URL accuracy, match evidence, and contact details.",
  ].join("\n");
}

