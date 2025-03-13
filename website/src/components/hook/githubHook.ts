import { useContext } from "react";
import { GithubContext } from "../context/githubContext";

export const useGithub = () => {
  const context = useContext(GithubContext);
  if (context === undefined) {
    throw new Error("useGithub must be used within a GithubProvider");
  }
  return context;
};
