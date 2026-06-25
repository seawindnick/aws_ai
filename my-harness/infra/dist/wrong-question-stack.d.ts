import * as cdk from 'aws-cdk-lib';
import { Construct } from 'constructs';
export interface WrongQuestionStackProps extends cdk.StackProps {
    envName?: string;
}
export declare class WrongQuestionStack extends cdk.Stack {
    constructor(scope: Construct, id: string, props?: WrongQuestionStackProps);
}
