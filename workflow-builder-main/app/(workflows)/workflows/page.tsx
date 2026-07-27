import WorkflowGrid from "../components/WorkflowGrid";
import GlobalLayout from "@/components/globals/GlobalLayout";
import WorkflowListingHeader from "../components/headers/workflowListingHeader";

export default function Home() {
  return (
    <>
      <WorkflowListingHeader />
      <GlobalLayout>
        <main className="w-full px-2 sm:px-4">
          <WorkflowGrid />
        </main>
      </GlobalLayout>
    </>
  );
}
