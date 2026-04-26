import { AppShell } from "@/components/app-shell";
import { RulesEditor } from "@/components/rules-editor";
import { getRulesData } from "@/lib/api";

export default async function RulesPage() {
  const { activeRuleDocument, quota } = await getRulesData();

  return (
    <AppShell eyebrow="Rule engine" title="Edit the ordered chain before the canvas exists." quota={quota}>
      <RulesEditor initialRules={activeRuleDocument} />
    </AppShell>
  );
}
