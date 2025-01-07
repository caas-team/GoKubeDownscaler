import React from "react";
import { GithubProvider } from "../components/context/githubContext";

const Root: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  return <GithubProvider>{children}</GithubProvider>;
};

export default Root;
